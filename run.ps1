param(
    [Parameter(Position = 0)]
    [string]$Command = "help"
)

$OTEL_DIR = "deployments/observability"
$LOCAL_DIR = "deployments/local"
$DOCKER_DIR = "deployments/docker"

switch ($Command) {

    "help" {
        Write-Host ""
        Write-Host "  anc-portal-be - Available Commands" -ForegroundColor Cyan
        Write-Host "  ===================================" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "  Development:" -ForegroundColor Yellow
        Write-Host "    .\run.ps1 dev           Run API with hot-reload (air)"
        Write-Host "    .\run.ps1 build         Build API binary"
        Write-Host "    .\run.ps1 test          Run all tests"
        Write-Host "    .\run.ps1 test-cover    Run tests with coverage report"
        Write-Host "    .\run.ps1 lint          Run golangci-lint"
        Write-Host "    .\run.ps1 ci            Run full CI pipeline (lint+test+vuln+build)"
        Write-Host "                             Set DISCORD_WEBHOOK_URL or .env.local to notify Discord" -ForegroundColor DarkGray
        Write-Host "    .\run.ps1 swagger       Generate Swagger docs (swag init)"
        Write-Host "    .\run.ps1 tidy          go mod tidy"
        Write-Host "    .\run.ps1 clean         Remove build artifacts"
        Write-Host ""
        Write-Host "  Database:" -ForegroundColor Yellow
        Write-Host "    .\run.ps1 migrate       Run database migrations"
        Write-Host "    .\run.ps1 seed          Run data seeding"
        Write-Host "    .\run.ps1 import        Import CSV data (interactive)"
        Write-Host ""
        Write-Host "  Worker:" -ForegroundColor Yellow
        Write-Host "    .\run.ps1 worker        Run background worker"
        Write-Host ""
        Write-Host "  Observability (OTel + Grafana):" -ForegroundColor Yellow
        Write-Host "    .\run.ps1 otel-up       Start observability stack"
        Write-Host "    .\run.ps1 otel-down     Stop observability stack"
        Write-Host "    .\run.ps1 otel-down-v   Stop + remove volumes"
        Write-Host "    .\run.ps1 otel-logs     View stack logs"
        Write-Host "    .\run.ps1 otel-ps       Show running containers"
        Write-Host ""
        Write-Host "  Local Dev Stack (PostgreSQL + Redis + Kafka):" -ForegroundColor Yellow
        Write-Host "    .\run.ps1 local-up      Start local dependencies"
        Write-Host "    .\run.ps1 local-down    Stop local dependencies"
        Write-Host "    .\run.ps1 local-down-v  Stop + remove volumes"
        Write-Host "    .\run.ps1 local-logs    View stack logs"
        Write-Host "    .\run.ps1 local-ps      Show running containers"
        Write-Host ""
        Write-Host "  Docker Build:" -ForegroundColor Yellow
        Write-Host "    .\run.ps1 docker-build         Build API Docker image"
        Write-Host "    .\run.ps1 docker-build-worker  Build Worker Docker image"
        Write-Host ""
        Write-Host "  CI Testing:" -ForegroundColor Yellow
        Write-Host "    .\run.ps1 ci-test-inject <type>   Inject CI failure (lint|test|build)" -ForegroundColor DarkGray
        Write-Host "    .\run.ps1 ci-test-clean           Remove all injected failures" -ForegroundColor DarkGray
        Write-Host ""
    }

    # -- Development --

    "dev" {
        Write-Host "[dev] Starting API with hot-reload..." -ForegroundColor Green
        air -c .air.local.toml
    }
    "build" {
        Write-Host "[build] Building API binary..." -ForegroundColor Green
        go build -o ./tmp/main.exe ./cmd/api
    }
    "test" {
        Write-Host "[test] Running all tests..." -ForegroundColor Green
        go test ./...
    }
    "test-cover" {
        $threshold = 25  # ปัจจุบัน 28.9% → ตั้ง 25% แล้วค่อยเพิ่มทีละ 5%
        Write-Host "[test-cover] Running tests with coverage (threshold: ${threshold}%)..." -ForegroundColor Green
        Write-Host ""

        go test -coverprofile coverage.out -covermode atomic ./...
        if ($LASTEXITCODE -ne 0) {
            Write-Host "  Tests FAILED" -ForegroundColor Red
            exit 1
        }

        Write-Host ""
        Write-Host "  Coverage by package:" -ForegroundColor Cyan
        Write-Host "  --------------------" -ForegroundColor DarkGray
        $coverOutput = go tool cover -func coverage.out
        $coverOutput | ForEach-Object {
            if ($_ -match 'total:') {
                Write-Host "  $_" -ForegroundColor Yellow
            } else {
                Write-Host "  $_" -ForegroundColor DarkGray
            }
        }

        # Extract total percentage
        $totalLine = $coverOutput | Select-String 'total:'
        if ($totalLine -match '([\d\.]+)%') {
            $coverage = [double]$Matches[1]
            Write-Host ""
            if ($coverage -lt $threshold) {
                Write-Host "  FAIL: coverage ${coverage}% is below threshold ${threshold}%" -ForegroundColor Red
                exit 1
            } else {
                Write-Host "  PASS: coverage ${coverage}% meets threshold ${threshold}%" -ForegroundColor Green
            }
        }

        Write-Host ""
        Write-Host "  HTML report: go tool cover -html=coverage.out" -ForegroundColor DarkGray
    }
    "lint" {
        Write-Host "[lint] Running golangci-lint..." -ForegroundColor Green
        golangci-lint run ./...
    }
    "ci" {
        $startTime = Get-Date
        $steps = @(
            @{ Name = "Lint";  Cmd = { golangci-lint run ./... } },
            @{ Name = "Test";  Cmd = { go test -count 1 ./... } },
            @{ Name = "Vuln";  Cmd = { govulncheck ./... } },
            @{ Name = "Build"; Cmd = {
                go build -o ./tmp/main.exe    ./cmd/api
                go build -o ./tmp/worker.exe  ./cmd/worker
                go build -o ./tmp/migrate.exe ./cmd/migrate
                go build -o ./tmp/seed.exe    ./cmd/seed
                go build -o ./tmp/import.exe  ./cmd/import
            }}
        )

        Write-Host ""
        Write-Host "  CI Pipeline - ANC Portal Backend" -ForegroundColor Cyan
        Write-Host "  =================================" -ForegroundColor Cyan
        Write-Host ""

        $results = @()
        $failed = $false
        $failureDetails = ""

        for ($i = 0; $i -lt $steps.Count; $i++) {
            $step = $steps[$i]
            $num = $i + 1
            $total = $steps.Count
            $label = "[$num/$total] $($step.Name)"

            Write-Host "  $label ..." -ForegroundColor Yellow -NoNewline

            $sw = [System.Diagnostics.Stopwatch]::StartNew()
            try {
                $output = & $step.Cmd 2>&1
                if ($LASTEXITCODE -ne 0 -and $step.Name -ne "Vuln") { throw "exit code $LASTEXITCODE" }
                $sw.Stop()
                $sec = [math]::Round($sw.Elapsed.TotalSeconds, 1)
                $results += @{ Name = $step.Name; Status = "PASS"; Time = $sec }
                Write-Host "`r  $label " -ForegroundColor Yellow -NoNewline
                Write-Host "PASS" -ForegroundColor Green -NoNewline
                Write-Host " (${sec}s)"
            } catch {
                $sw.Stop()
                $sec = [math]::Round($sw.Elapsed.TotalSeconds, 1)
                $results += @{ Name = $step.Name; Status = "FAIL"; Time = $sec }
                Write-Host "`r  $label " -ForegroundColor Yellow -NoNewline
                Write-Host "FAIL" -ForegroundColor Red -NoNewline
                Write-Host " (${sec}s)"
                # Show failure details
                if ($output) {
                    Write-Host ""
                    Write-Host "  ── Failure Details ──" -ForegroundColor Red
                    $output | ForEach-Object { Write-Host "  $_" -ForegroundColor DarkGray }
                    Write-Host ""
                    # Capture for Discord (limit to 1000 chars to fit embed)
                    $rawDetails = ($output | Out-String).Trim()
                    if ($rawDetails.Length -gt 1000) {
                        $rawDetails = $rawDetails.Substring(0, 1000) + "..."
                    }
                    $failureDetails = $rawDetails
                }
                $failed = $true
                break
            }
        }

        # Remaining steps (skipped)
        for ($j = $results.Count; $j -lt $steps.Count; $j++) {
            $results += @{ Name = $steps[$j].Name; Status = "SKIP"; Time = 0 }
        }

        $totalSec = [math]::Round(((Get-Date) - $startTime).TotalSeconds, 1)

        # Summary table
        Write-Host ""
        Write-Host "  +---------+--------+---------+" -ForegroundColor DarkGray
        Write-Host "  | Step    | Status | Time    |" -ForegroundColor DarkGray
        Write-Host "  +---------+--------+---------+" -ForegroundColor DarkGray
        foreach ($r in $results) {
            $sColor = "DarkGray"
            if ($r.Status -eq "PASS") { $sColor = "Green" }
            if ($r.Status -eq "FAIL") { $sColor = "Red" }
            $nameCol = $r.Name.PadRight(7)
            $statCol = $r.Status.PadRight(6)
            if ($r.Time -gt 0) { $timeCol = ("" + $r.Time + "s").PadRight(7) } else { $timeCol = "  -    " }
            Write-Host "  | " -ForegroundColor DarkGray -NoNewline
            Write-Host $nameCol -NoNewline
            Write-Host " | " -ForegroundColor DarkGray -NoNewline
            Write-Host $statCol -ForegroundColor $sColor -NoNewline
            Write-Host " | " -ForegroundColor DarkGray -NoNewline
            Write-Host $timeCol -NoNewline
            Write-Host " |" -ForegroundColor DarkGray
        }
        Write-Host "  +---------+--------+---------+" -ForegroundColor DarkGray

        # Final verdict
        Write-Host ""
        if ($failed) {
            Write-Host "  PIPELINE FAILED" -ForegroundColor Red -NoNewline
        } else {
            Write-Host "  PIPELINE PASSED" -ForegroundColor Green -NoNewline
        }
        Write-Host " (total: ${totalSec}s)"
        Write-Host ""

        # ── Discord notification ──
        $webhookUrl = $env:DISCORD_WEBHOOK_URL
        # Fallback: อ่านจาก .env.local ถ้ามี
        if (-not $webhookUrl -and (Test-Path ".env.local")) {
            $line = Get-Content ".env.local" | Where-Object { $_ -match "^DISCORD_WEBHOOK_URL=" } | Select-Object -First 1
            if ($line) { $webhookUrl = ($line -split "=", 2)[1].Trim('"').Trim("'") }
        }

        if ($webhookUrl) {
            Write-Host "  Sending to Discord..." -ForegroundColor DarkGray -NoNewline

            # Git info
            $branch = git rev-parse --abbrev-ref HEAD 2>$null
            if (-not $branch) { $branch = "unknown" }
            $sha = git rev-parse --short HEAD 2>$null
            if (-not $sha) { $sha = "unknown" }
            $commitMsg = git log -1 --pretty=%s 2>$null
            if (-not $commitMsg) { $commitMsg = "" }
            if ($commitMsg.Length -gt 72) { $commitMsg = $commitMsg.Substring(0, 72) + "..." }
            $commitMsg = $commitMsg -replace '\\', '\\\\' -replace '"', '\"'
            $hostname = $env:COMPUTERNAME
            $username = $env:USERNAME
            $repoUrl = git config --get remote.origin.url 2>$null
            if ($repoUrl) {
                $repoUrl = $repoUrl -replace '\.git$', ''
                $repoUrl = $repoUrl -replace 'git@github\.com:', 'https://github.com/'
            }

            # Status
            if ($failed) {
                $titleEmoji = "\u274c"
                $title = "Pipeline Failed"
                $color = 15158332
                $statusBadge = "FAILED"
            } else {
                $titleEmoji = "\u2705"
                $title = "Pipeline Passed"
                $color = 3066993
                $statusBadge = "PASSED"
            }

            # Job status table (infra-style)
            $jobLines = @()
            foreach ($r in $results) {
                switch ($r.Status) {
                    "PASS" { $icon = "\u2705" }
                    "FAIL" { $icon = "\u274c" }
                    "SKIP" { $icon = "\u23ed\ufe0f" }
                    default { $icon = "\u2b1c" }
                }
                $t = if ($r.Time -gt 0) { "``$($r.Time)s``" } else { "``-``" }
                $jobLines += "${icon} **$($r.Name)** $t"
            }
            $jobsText = $jobLines -join '\n'

            # Failure details
            $failureField = ""
            $locationField = ""
            if ($failed -and $failureDetails) {
                $escapedDetails = $failureDetails -replace '\\', '\\\\' -replace '"', '\"' -replace "`r`n", '\n' -replace "`n", '\n' -replace "`t", '  '
                $failureField = ",{`"name`":`"\u26a0\ufe0f Failure Output`",`"value`":`"``````\n${escapedDetails}\n```````",`"inline`":false}"

                $locMatches = [regex]::Matches($failureDetails, '(?m)([a-zA-Z0-9_\-\\\/\.]+\.go:\d+(?::\d+)?)')
                $locations = @()
                foreach ($m in $locMatches) {
                    $loc = $m.Value -replace '\\', '/'
                    if ($locations -notcontains $loc) { $locations += $loc }
                }
                if ($locations.Count -gt 0 -and $repoUrl) {
                    $shownLocs = $locations | Select-Object -First 8
                    $locLines = ($shownLocs | ForEach-Object {
                        $fileLine = $_
                        if ($fileLine -match '^(.+):(\d+)') {
                            $fPath = $Matches[1]
                            $lineNum = $Matches[2]
                            $linkUrl = "${repoUrl}/blob/${branch}/${fPath}#L${lineNum}"
                            "\u2022 [``${fileLine}``](${linkUrl})"
                        } else {
                            "\u2022 ``${fileLine}``"
                        }
                    }) -join '\n'
                    if ($locations.Count -gt 8) {
                        $extra = $locations.Count - 8
                        $locLines += '\n... +' + $extra + ' more'
                    }
                    $locationField = ",{`"name`":`"\ud83d\udccd Error Locations`",`"value`":`"${locLines}`",`"inline`":false}"
                }
            }

            # Build description block (infra-style metadata)
            $desc = "``````env\nENV      = local\nSTATUS   = ${statusBadge}\nDURATION = ${totalSec}s\nMACHINE  = ${username}@${hostname}\n``````"

            # Commit URL
            $commitField = "``${sha}``"
            if ($repoUrl) {
                $fullSha = git rev-parse HEAD 2>$null
                if ($fullSha) {
                    $commitUrl = "${repoUrl}/commit/${fullSha}"
                    $commitField = "[``${sha}``](${commitUrl})"
                }
            }

            $ts = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

            $payload = @"
{"embeds":[{"title":"$titleEmoji $title","description":"$desc","color":$color,"fields":[{"name":"\ud83d\udd00 Branch","value":"``$branch``","inline":true},{"name":"\ud83d\udcdd Commit","value":"$commitField","inline":true},{"name":"\ud83d\udc64 Author","value":"${username}","inline":true},{"name":"\ud83d\udcac Message","value":"$commitMsg","inline":false},{"name":"\ud83d\udee0\ufe0f Jobs","value":"${jobsText}","inline":false}${failureField}${locationField}],"footer":{"text":"\ud83d\udce1 ANC Portal CI \u2022 Local"},"timestamp":"$ts"}]}
"@
            $utf8 = [System.Text.Encoding]::UTF8.GetBytes($payload)

            try {
                Invoke-RestMethod -Uri $webhookUrl -Method Post -ContentType "application/json; charset=utf-8" -Body $utf8 -ErrorAction Stop | Out-Null
                Write-Host " sent!" -ForegroundColor Green
            } catch {
                Write-Host " failed: $($_.Exception.Message)" -ForegroundColor Red
            }
        }
    }
    "tidy" {
        Write-Host "[tidy] Running go mod tidy..." -ForegroundColor Green
        go mod tidy
    }
    "swagger" {
        Write-Host "[swagger] Generating Swagger docs..." -ForegroundColor Green
        swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
        Write-Host "Done. Open http://localhost:3000/swagger/index.html" -ForegroundColor Green
    }
    "clean" {
        Write-Host "[clean] Removing build artifacts..." -ForegroundColor Green
        if (Test-Path tmp) { Remove-Item -Recurse -Force tmp }
        Write-Host "Done." -ForegroundColor Green
    }

    # -- Database --

    "migrate" {
        Write-Host "[migrate] Running database migrations..." -ForegroundColor Green
        go run ./cmd/migrate
    }
    "seed" {
        Write-Host "[seed] Running data seeding..." -ForegroundColor Green
        go run ./cmd/seed
    }
    "import" {
        Write-Host "[import] Import CSV data into database" -ForegroundColor Green
        $envFile = Read-Host "  Enter env file path (e.g. .env.local)"
        $csvPath = Read-Host "  Enter CSV file path (e.g. .\base_data\users.csv)"
        $svcType = Read-Host "  Enter service_type (insurer, insurer_installment, province, user)"
        go run ./cmd/import/main.go --env $envFile --path $csvPath --service_type $svcType
    }

    # -- Worker --

    "worker" {
        Write-Host "[worker] Starting background worker..." -ForegroundColor Green
        go run ./cmd/worker
    }

    # -- Observability (OTel + Grafana) --

    "otel-up" {
        Write-Host "[otel] Starting observability stack..." -ForegroundColor Green
        docker compose -f "$OTEL_DIR/docker-compose.yaml" --env-file "$OTEL_DIR/.env" up -d
    }
    "otel-down" {
        Write-Host "[otel] Stopping observability stack..." -ForegroundColor Green
        docker compose -f "$OTEL_DIR/docker-compose.yaml" --env-file "$OTEL_DIR/.env" down
    }
    "otel-down-v" {
        Write-Host "[otel] Stopping + removing volumes..." -ForegroundColor Yellow
        docker compose -f "$OTEL_DIR/docker-compose.yaml" --env-file "$OTEL_DIR/.env" down -v
    }
    "otel-logs" {
        docker compose -f "$OTEL_DIR/docker-compose.yaml" logs -f
    }
    "otel-ps" {
        docker compose -f "$OTEL_DIR/docker-compose.yaml" ps
    }

    # -- Local Dev Stack (PostgreSQL + Redis + Kafka) --

    "local-up" {
        Write-Host "[local] Starting local dependencies..." -ForegroundColor Green
        docker compose -f "$LOCAL_DIR/docker-compose.yaml" --env-file "$LOCAL_DIR/.env" up -d
    }
    "local-down" {
        Write-Host "[local] Stopping local dependencies..." -ForegroundColor Green
        docker compose -f "$LOCAL_DIR/docker-compose.yaml" --env-file "$LOCAL_DIR/.env" down
    }
    "local-down-v" {
        Write-Host "[local] Stopping + removing volumes..." -ForegroundColor Yellow
        docker compose -f "$LOCAL_DIR/docker-compose.yaml" --env-file "$LOCAL_DIR/.env" down -v
    }
    "local-logs" {
        docker compose -f "$LOCAL_DIR/docker-compose.yaml" logs -f
    }
    "local-ps" {
        docker compose -f "$LOCAL_DIR/docker-compose.yaml" ps
    }

    # -- Docker Build --

    "docker-build" {
        Write-Host "[docker] Building API image..." -ForegroundColor Green
        docker build -f "$DOCKER_DIR/Dockerfile" -t anc-portal-be .
    }
    "docker-build-worker" {
        Write-Host "[docker] Building Worker image (multi-target)..." -ForegroundColor Green
        docker build -f "$DOCKER_DIR/Dockerfile" --target worker -t anc-portal-worker .
    }

    # -- CI Testing (inject failures to verify pipeline) --

    "ci-test-inject" {
        $type = $args[0]
        if (-not $type) {
            Write-Host ""
            Write-Host "  Usage: .\run.ps1 ci-test-inject <type>" -ForegroundColor Yellow
            Write-Host ""
            Write-Host "  Types:" -ForegroundColor Cyan
            Write-Host "    lint    Inject gosec G101 (hardcoded credential) -> Lint FAIL"
            Write-Host "    test    Inject failing unit test (wrong calc)    -> Test FAIL"
            Write-Host "    build   Inject compile error (undefined var)    -> Build FAIL"
            Write-Host ""
            Write-Host "  After inject: push to trigger CI, then run ci-test-clean" -ForegroundColor DarkGray
            Write-Host ""
            exit 0
        }

        switch ($type) {
            "lint" {
                $src = "testdata/ci/inject_lint_fail.go.bak"
                $dst = "internal/shared/dto/ci_test_lint_fail.go"
                if (-not (Test-Path $src)) { Write-Host "  ERROR: $src not found" -ForegroundColor Red; exit 1 }
                Copy-Item $src $dst
                Write-Host ""
                Write-Host "  Injected: $dst" -ForegroundColor Yellow
                Write-Host "  Expected: Lint step -> FAIL (gosec G101: hardcoded credentials)" -ForegroundColor DarkGray
                Write-Host "  Cleanup:  .\run.ps1 ci-test-clean" -ForegroundColor DarkGray
                Write-Host ""
            }
            "test" {
                $src = "testdata/ci/inject_test_fail.go.bak"
                $dst = "internal/shared/utils/ci_test_fail_test.go"
                if (-not (Test-Path $src)) { Write-Host "  ERROR: $src not found" -ForegroundColor Red; exit 1 }
                Copy-Item $src $dst
                Write-Host ""
                Write-Host "  Injected: $dst" -ForegroundColor Yellow
                Write-Host "  Expected: Test step -> FAIL (calculateDiscount wrong result)" -ForegroundColor DarkGray
                Write-Host "  Cleanup:  .\run.ps1 ci-test-clean" -ForegroundColor DarkGray
                Write-Host ""
            }
            "build" {
                $src = "testdata/ci/inject_build_fail.go.bak"
                $dst = "internal/shared/utils/ci_test_build_fail.go"
                if (-not (Test-Path $src)) { Write-Host "  ERROR: $src not found" -ForegroundColor Red; exit 1 }
                Copy-Item $src $dst
                Write-Host ""
                Write-Host "  Injected: $dst" -ForegroundColor Yellow
                Write-Host "  Expected: Build step -> FAIL (undefined variable)" -ForegroundColor DarkGray
                Write-Host "  Cleanup:  .\run.ps1 ci-test-clean" -ForegroundColor DarkGray
                Write-Host ""
            }
            default {
                Write-Host "  Unknown type: $type (use: lint, test, build)" -ForegroundColor Red
            }
        }
    }
    "ci-test-clean" {
        Write-Host ""
        Write-Host "  Cleaning injected CI test files..." -ForegroundColor Cyan
        $files = @(
            "internal/shared/dto/ci_test_lint_fail.go",
            "internal/shared/utils/ci_test_fail_test.go",
            "internal/shared/utils/ci_test_build_fail.go"
        )
        $removed = 0
        foreach ($f in $files) {
            if (Test-Path $f) {
                Remove-Item $f
                Write-Host "  Removed: $f" -ForegroundColor Green
                $removed++
            }
        }
        if ($removed -eq 0) {
            Write-Host "  Nothing to clean - no injected files found" -ForegroundColor DarkGray
        } else {
            Write-Host ""
            Write-Host "  Cleaned $removed file(s). Safe to push for CI success test." -ForegroundColor Green
        }
        Write-Host ""
    }

    # -- Unknown --

    default {
        Write-Host "Unknown command: $Command" -ForegroundColor Red
        Write-Host "Run: .\run.ps1 help" -ForegroundColor Yellow
    }
}
