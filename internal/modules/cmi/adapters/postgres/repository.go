package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/domain"
	"github.com/onizukazaza/anc-portal-be-fake/pkg/log"
)

// CMIPolicyRepository reads CMI policy data from the external (mpk) database.
type CMIPolicyRepository struct {
	pool *pgxpool.Pool
}

func NewCMIPolicyRepository(pool *pgxpool.Pool) *CMIPolicyRepository {
	return &CMIPolicyRepository{pool: pool}
}

// JobExists ตรวจสอบว่า job มีอยู่จริง — ใช้ EXISTS เพื่อ short-circuit เร็วกว่า COUNT
func (r *CMIPolicyRepository) JobExists(ctx context.Context, jobID string) (bool, error) {
	ctx, span := appOtel.Tracer(appOtel.TracerCMIRepo).Start(ctx, "JobExists")
	defer span.End()
	span.SetAttributes(attribute.String("job_id", jobID))

	log.L().Info().Str("layer", "repository").Str("job_id", jobID).Msg("→ CMI JobExists")

	const q = `SELECT EXISTS(SELECT 1 FROM job WHERE id = $1)`

	var exists bool
	if err := r.pool.QueryRow(ctx, q, jobID).Scan(&exists); err != nil {
		log.L().Error().Err(err).Str("layer", "repository").Str("job_id", jobID).Msg("← CMI JobExists error")
		return false, err
	}
	log.L().Info().Str("layer", "repository").Str("job_id", jobID).Bool("exists", exists).Msg("← CMI JobExists done")
	return exists, nil
}

// ===================================================================
// SQL Fragments — แยก query ออกเป็นส่วนย่อยเพื่อให้ maintain ง่าย
// ===================================================================

// sqlSelectJobFields — scalar columns จาก job + is_problem flag
func sqlSelectJobFields() string {
	return `
		j.id,
		j.job_type,
		COALESCE(j.job_ref_id, ''),
		j.status,
		COALESCE(j.agent_id, ''),
		COALESCE(j.commission_tax_type, ''),
		COALESCE(j.with_holding_tax, false),
		COALESCE(agent_.tax_allocation, false),
		EXISTS(
			SELECT 1 FROM job_log jl
			WHERE jl.job_id = j.id
			  AND jl.status IN ('waiting','resolved','open')
		),
		(p_info.package #>> '{cmi_package, insurer_id}')::int,
		i_ins.name,
		COALESCE(p.id::varchar, '')`
}

// sqlMotorInfo — motor_info jsonb object
func sqlMotorInfo() string {
	return `
		jsonb_build_object(
			'id',             msm.id,
			'year',           m.year,
			'brand',          mb.brand,
			'brand_logo_url', mb.logo_url,
			'model',          mm.model,
			'sub_model',      msm.detail,
			'motor_code',     msm.motor_code
		)`
}

// sqlAssetInfo — asset_info jsonb object
func sqlAssetInfo() string {
	return `
		jsonb_build_object(
			'id',        a2.id,
			'cmi_code',  j.cmi_code,
			'license_plate', jsonb_build_object(
				'car_registration', a2.description #>> '{car_registration}',
				'province_id',      p5.id,
				'province_name',    COALESCE(p5.name_th, '')
			),
			'chassis_no', a2.description #>> '{chassis_no}',
			'engine_no',  a2.description #>> '{engine_no}',
			'color', jsonb_build_object(
				'color_id',      (a2.description #>> '{color,color_id}')::int,
				'color_code',    a2.description #>> '{color,color_code}',
				'color_name_th', a2.description #>> '{color,color_name_th}',
				'color_name_en', a2.description #>> '{color,color_name_en}'
			)
		)`
}

// sqlInsured — insured json object
func sqlInsured() string {
	return `
		json_build_object(
			'id',           i.id,
			'type',         i.type,
			'prefix_id',    i.prefix_id,
			'prefix',       ip1.full_th,
			'first_name',   i.first_name,
			'last_name',    i.last_name,
			'citizen_id',   i.citizen_id,
			'passport_id',  i.passport_id,
			'gender',       i.gender::varchar,
			'birth_date',   i.birth_date,
			'phone_number', i.phone_number,
			'email',        i.email
		)`
}

// sqlPolicyDates — policy dates jsonb object
func sqlPolicyDates() string {
	return `
		jsonb_build_object(
			'cmi', jsonb_build_object(
				'start_date', COALESCE(to_char(p4.started_date, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
				'end_date',   COALESCE(to_char(p4.expired_date, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
			)
		)`
}

// sqlAddressSubquery — reusable address subquery for a given address type
func sqlAddressSubquery(alias, addrType string) string {
	return fmt.Sprintf(`(SELECT row_to_json(x) FROM (
		SELECT %[1]s.address, %[1]s.postcode, %[1]s.province_id, pv_%[1]s.name_th AS province_name,
		       %[1]s.district_id, dt_%[1]s.name_th AS district_name, %[1]s.sub_district_id, sd_%[1]s.name_th AS sub_district_name,
		       %[1]s.moo, %[1]s.village, %[1]s.road, %[1]s.soi, %[1]s.first_name, %[1]s.last_name, %[1]s.phone_number
		FROM address %[1]s
		LEFT JOIN province pv_%[1]s ON pv_%[1]s.id = %[1]s.province_id
		LEFT JOIN district dt_%[1]s ON dt_%[1]s.id = %[1]s.district_id
		LEFT JOIN sub_district sd_%[1]s ON sd_%[1]s.id = %[1]s.sub_district_id
		WHERE %[1]s.insured_id = i.id AND %[1]s.address_type = '%[2]s'
		LIMIT 1
	) x)`, alias, addrType)
}

// sqlAddressSet — address set (main / shipping / billing) jsonb object
func sqlAddressSet() string {
	return fmt.Sprintf(`
		jsonb_build_object(
			'main_address',     %s,
			'shipping_address', %s,
			'billing_address',  %s
		)`,
		sqlAddressSubquery("a_m", "main"),
		sqlAddressSubquery("a_s", "shipping"),
		sqlAddressSubquery("a_b", "billing"),
	)
}

// sqlAgentInfo — agent info jsonb object
func sqlAgentInfo() string {
	return `
		jsonb_build_object(
			'id',                au.id,
			'first_name',        au.first_name,
			'last_name',         au.last_name,
			'nick_name',         au.nick_name,
			'email',             au.email,
			'profile_image_url', au.profile_image_url,
			'role',              COALESCE(au.role::text, ''),
			'class',             agent_.class,
			'anc_agent_id',      agent_.anc_agent_id,
			'organization_name', org.name,
			'team_name',         CASE WHEN team.lobby THEN 'ล๊อบบี้' ELSE team.name END,
			'team_role',         member.role,
			'status',            au.status
		)`
}

// sqlProducts — aggregated products jsonb array
func sqlProducts() string {
	return `
		COALESCE((
			SELECT jsonb_agg(
				jsonb_build_object('product_id', pr.id, 'product', COALESCE(pr.package, '{}'::jsonb))
				ORDER BY COALESCE((pr.package #>> '{list_index}')::int, 999)
			)
			FROM quotation_product qp2
			INNER JOIN product pr ON pr.id = qp2.product_id
			WHERE qp2.quotation_id = q.id
		), '[]'::jsonb)`
}

// sqlPayments — aggregated payments jsonb array
func sqlPayments() string {
	return `
		COALESCE((
			SELECT jsonb_agg(jsonb_build_object(
				'id',        pp.id,
				'method',    pp.method,
				'amount',    pp.amount,
				'currency',  pp.currency,
				'status',    COALESCE(pp.status::varchar, ''),
				'ref',       pp.ref,
				'paid_timestamp', pp.paid_datetime,
				'created_datetime', to_char(pp.created_datetime::timestamp, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
			) ORDER BY pp.created_datetime DESC)
			FROM payment_purchase pp
			WHERE pp.job_id = j.id
			  AND (CASE WHEN pp.method IN ('qr','card') THEN pp.have_payment_notify AND pp.status::varchar = 'approved' ELSE true END)
		), '[]'::jsonb)`
}

// sqlInsuranceDocs — insurance documents jsonb array
func sqlInsuranceDocs() string {
	return `
		COALESCE((
			SELECT jsonb_agg(jsonb_build_object(
				'title',        COALESCE(att.file_title, ''),
				'description',  COALESCE(att.file_description, ''),
				'document_url', COALESCE(att.url, '')
			))
			FROM attachment att
			WHERE att.job_id = j.id
			  AND att.insured_id IS NULL
			  AND att.url IS NOT NULL AND att.url <> ''
			  AND (
			      att.file_description SIMILAR TO '(car_registration|citizen|old_policy|company_registration|request_transfer_code)'
			      OR att.file_description LIKE 'other_%%'
			      OR att.file_description LIKE 'internal_other_%%'
			  )
		), '[]'::jsonb)`
}

// sqlInsuredDocs — insured documents jsonb array
func sqlInsuredDocs() string {
	return `
		COALESCE((
			SELECT jsonb_agg(jsonb_build_object(
				'title',        att2.file_title,
				'description',  att2.file_description,
				'document_url', att2.url
			))
			FROM attachment att2
			WHERE att2.insured_id = i.id
			  AND att2.file_description SIMILAR TO '(citizen|driving_license|house_registration|company_registration)'
		), '[]'::jsonb)`
}

// sqlQuoteInfo — quote info jsonb object
func sqlQuoteInfo() string {
	return `
		jsonb_build_object(
			'id',           q.id,
			'job_id',       q.job_id,
			'download_url', COALESCE(atta_dl.download_ref, ''),
			'image_url',    COALESCE(atta_img.download_ref, ''),
			'created_at',   to_char(q.created_datetime, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			'updated_at',   to_char(q.updated_datetime, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		)`
}

// sqlFromJoins — FROM + LEFT JOIN clauses
func sqlFromJoins() string {
	return `
		FROM job j
		LEFT JOIN quotation q        ON q.id = j.quotation_id
		LEFT JOIN attachment atta_dl  ON atta_dl.job_id = j.id  AND atta_dl.file_description = 'quotations'
		LEFT JOIN attachment atta_img ON atta_img.job_id = j.id AND atta_img.file_description = 'quotations-image'
		LEFT JOIN quotation_product qp ON qp.quotation_id = q.id
		LEFT JOIN product p            ON p.id = qp.product_id
		LEFT JOIN insured i            ON i.id = q.insured_id
		LEFT JOIN prefix ip1           ON ip1.id = i.prefix_id
		LEFT JOIN asset a2             ON a2.id = j.asset_id
		LEFT JOIN motor m              ON m.id = a2.motor_id
		LEFT JOIN motor_brand mb       ON mb.id = m.brand_id
		LEFT JOIN motor_model mm       ON mm.id = m.model_id
		LEFT JOIN motor_sub_model msm  ON msm.id = m.sub_model_id
		LEFT JOIN province p5          ON p5.id = a2.asset_province
		LEFT JOIN policy p4            ON p4.job_id = j.id AND p4.insurance_type = 'cmi'
		LEFT JOIN "user" au            ON au.id = j.agent_id
		LEFT JOIN agent agent_         ON agent_.id = au.id
		LEFT JOIN member               ON member.agent_id = j.agent_id AND member.status = 'active'
		LEFT JOIN organization org     ON org.id = member.organization_id
		LEFT JOIN team                 ON team.id = member.team_id
		LEFT JOIN product p_info       ON p_info.id = (
			SELECT qp3.product_id FROM quotation_product qp3
			WHERE qp3.quotation_id = j.quotation_id
			ORDER BY qp3.created_datetime DESC LIMIT 1
		)
		LEFT JOIN insurer i_ins        ON i_ins.id = (p_info.package #>> '{cmi_package, insurer_id}')::int
		WHERE j.id = $1
		LIMIT 1`
}

// buildFindPolicyQuery assembles the full CMI policy query from fragments.
func buildFindPolicyQuery() string {
	return fmt.Sprintf(`SELECT
		%s,
		%s,
		%s,
		%s,
		%s,
		%s,
		%s,
		%s,
		%s,
		%s,
		%s,
		%s,
		j.created_datetime,
		j.updated_datetime
		%s`,
		sqlSelectJobFields(),
		sqlMotorInfo(),
		sqlAssetInfo(),
		sqlInsured(),
		sqlPolicyDates(),
		sqlAddressSet(),
		sqlAgentInfo(),
		sqlProducts(),
		sqlPayments(),
		sqlInsuranceDocs(),
		sqlInsuredDocs(),
		sqlQuoteInfo(),
		sqlFromJoins(),
	)
}

// ===================================================================
// FindPolicyByJobID — ดึงข้อมูล CMI policy ครบจบใน 1 query
// ===================================================================

// FindPolicyByJobID ดึงข้อมูล CMI policy ครบจบใน 1 query
// ใช้ json_build_object เพื่อลด round-trip และ serialize ใน DB layer
func (r *CMIPolicyRepository) FindPolicyByJobID(ctx context.Context, jobID string) (*domain.CMIPolicy, error) {
	ctx, span := appOtel.Tracer(appOtel.TracerCMIRepo).Start(ctx, "FindPolicyByJobID")
	defer span.End()
	span.SetAttributes(attribute.String("job_id", jobID))

	log.L().Info().Str("layer", "repository").Str("job_id", jobID).Msg("→ CMI FindPolicyByJobID")

	q := buildFindPolicyQuery()

	pol, err := scanCMIPolicy(r.pool.QueryRow(ctx, q, jobID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	log.L().Info().Str("layer", "repository").Str("job_id", jobID).Msg("← CMI FindPolicyByJobID OK")
	return pol, nil
}

// scanCMIPolicy scans a single row into a CMIPolicy, unmarshalling embedded JSON.
func scanCMIPolicy(row pgx.Row) (*domain.CMIPolicy, error) {
	var (
		pol             domain.CMIPolicy
		motorJSON       []byte
		assetJSON       []byte
		insuredJSON     []byte
		policyJSON      []byte
		addressJSON     []byte
		agentJSON       []byte
		productsJSON    []byte
		paymentsJSON    []byte
		docsJSON        []byte
		insuredDocsJSON []byte
		quoteJSON       []byte
	)

	err := row.Scan(
		&pol.JobID,
		&pol.JobType,
		&pol.JobRefID,
		&pol.JobStatus,
		&pol.AgentID,
		&pol.CommissionTaxType,
		&pol.WithHoldingTax,
		&pol.TaxAllocation,
		&pol.IsProblem,
		&pol.InsurerID,
		&pol.InsurerName,
		&pol.ProductID,
		&motorJSON,
		&assetJSON,
		&insuredJSON,
		&policyJSON,
		&addressJSON,
		&agentJSON,
		&productsJSON,
		&paymentsJSON,
		&docsJSON,
		&insuredDocsJSON,
		&quoteJSON,
		&pol.CreatedAt,
		&pol.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Unmarshal embedded JSON objects
	if err := unmarshalIfNotNil(motorJSON, &pol.Motor); err != nil {
		return nil, fmt.Errorf("unmarshal motor: %w", err)
	}
	if err := unmarshalIfNotNil(assetJSON, &pol.Asset); err != nil {
		return nil, fmt.Errorf("unmarshal asset: %w", err)
	}
	if err := unmarshalIfNotNil(insuredJSON, &pol.Insured); err != nil {
		return nil, fmt.Errorf("unmarshal insured: %w", err)
	}
	if err := unmarshalIfNotNil(policyJSON, &pol.Policy); err != nil {
		return nil, fmt.Errorf("unmarshal policy: %w", err)
	}
	if err := unmarshalIfNotNil(addressJSON, &pol.Address); err != nil {
		return nil, fmt.Errorf("unmarshal address: %w", err)
	}
	if err := unmarshalIfNotNil(agentJSON, &pol.Agent); err != nil {
		return nil, fmt.Errorf("unmarshal agent: %w", err)
	}
	if err := unmarshalIfNotNil(quoteJSON, &pol.Quote); err != nil {
		return nil, fmt.Errorf("unmarshal quote: %w", err)
	}

	pol.Products = productsJSON
	pol.Payments = paymentsJSON
	pol.Documents = docsJSON
	pol.InsuredDocuments = insuredDocsJSON

	return &pol, nil
}

func unmarshalIfNotNil(data []byte, dest any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, dest)
}
