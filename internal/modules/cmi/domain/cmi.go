package domain

import (
	"encoding/json"
	"time"
)

// CMIPolicy คือข้อมูลงาน พรบ. เดี่ยว (Compulsory Motor Insurance)
type CMIPolicy struct {
	JobID             string          `json:"job_id"`
	JobType           string          `json:"job_type"`
	JobRefID          string          `json:"job_ref_id"`
	JobStatus         string          `json:"job_status"`
	AgentID           string          `json:"agent_id"`
	CommissionTaxType string          `json:"commission_tax_type"`
	WithHoldingTax    bool            `json:"with_holding_tax"`
	TaxAllocation     bool            `json:"tax_allocation"`
	IsProblem         bool            `json:"is_problem"`
	InsurerID         *int            `json:"insurer_id"`
	InsurerName       *string         `json:"insurer_name"`
	ProductID         string          `json:"product_id"`
	Motor             *MotorInfo      `json:"motor_info"`
	Asset             *AssetInfo      `json:"asset_info"`
	Insured           *InsuredInfo    `json:"insured"`
	Policy            *PolicyDate     `json:"policy"`
	Address           *AddressSet     `json:"address_info"`
	Agent             *AgentInfo      `json:"agent_info"`
	Products          json.RawMessage `json:"products"`
	Payments          json.RawMessage `json:"payment_info"`
	Documents         json.RawMessage `json:"insurance_documents"`
	InsuredDocuments  json.RawMessage `json:"insured_documents"`
	Quote             *QuoteInfo      `json:"quote"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// MotorInfo ข้อมูลรถยนต์ (ยี่ห้อ/รุ่น/ปี)
type MotorInfo struct {
	ID       *int   `json:"id"`
	Year     string `json:"year"`
	Brand    string `json:"brand"`
	LogoURL  string `json:"brand_logo_url"`
	Model    string `json:"model"`
	SubModel string `json:"sub_model"`
	Code     string `json:"motor_code"`
}

// AssetInfo ข้อมูลรถเพิ่มเติม (ทะเบียน/เลขตัวถัง)
type AssetInfo struct {
	ID           string        `json:"id"`
	CMICode      string        `json:"cmi_code"`
	LicensePlate *LicensePlate `json:"license_plate"`
	ChassisNo    string        `json:"chassis_no"`
	EngineNo     string        `json:"engine_no"`
	Color        *CarColor     `json:"color"`
}

type LicensePlate struct {
	Registration string `json:"car_registration"`
	ProvinceID   *int   `json:"province_id"`
	ProvinceName string `json:"province_name"`
}

type CarColor struct {
	ID     *int   `json:"color_id"`
	Code   string `json:"color_code"`
	NameTH string `json:"color_name_th"`
	NameEN string `json:"color_name_en"`
}

// InsuredInfo ข้อมูลผู้เอาประกัน
type InsuredInfo struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	PrefixID    *int   `json:"prefix_id"`
	Prefix      string `json:"prefix"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	CitizenID   string `json:"citizen_id"`
	PassportID  string `json:"passport_id"`
	Gender      string `json:"gender"`
	BirthDate   string `json:"birth_date"`
	PhoneNumber string `json:"phone_number"`
	Email       string `json:"email"`
}

// PolicyDate วันที่คุ้มครอง พรบ.
type PolicyDate struct {
	CMI *DateRange `json:"cmi"`
}

type DateRange struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// AddressSet ชุดที่อยู่ (main/shipping/billing)
type AddressSet struct {
	MainAddress     json.RawMessage `json:"main_address"`
	ShippingAddress json.RawMessage `json:"shipping_address"`
	BillingAddress  json.RawMessage `json:"billing_address"`
}

// AgentInfo ข้อมูลนายหน้า
type AgentInfo struct {
	ID              string `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	NickName        string `json:"nick_name"`
	Email           string `json:"email"`
	ProfileImageURL string `json:"profile_image_url"`
	Role            string `json:"role"`
	Class           string `json:"class"`
	AncAgentID      string `json:"anc_agent_id"`
	OrgName         string `json:"organization_name"`
	TeamName        string `json:"team_name"`
	TeamRole        string `json:"team_role"`
	Status          string `json:"status"`
}

// QuoteInfo ข้อมูลใบเสนอราคา
type QuoteInfo struct {
	ID          string `json:"id"`
	JobID       string `json:"job_id"`
	DownloadURL string `json:"download_url"`
	ImageURL    string `json:"image_url"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
