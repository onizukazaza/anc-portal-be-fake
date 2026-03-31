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
            if ($commitMsg.Length -gt 80) { $commitMsg = $commitMsg.Substring(0, 80) + "..." }
            # Escape special JSON characters in commit message
            $commitMsg = $commitMsg -replace '\\', '\\\\' -replace '"', '\"'
            $hostname = $env:COMPUTERNAME
            $username = $env:USERNAME

            # Status (use Unicode escapes for Discord)
            if ($failed) {
                $titleEmoji = "\u274c"
                $title = "CI Pipeline Failed (Local)"
                $color = 15158332
            } else {
                $titleEmoji = "\u2705"
                $title = "CI Pipeline Passed (Local)"
                $color = 3066993
            }

            # Job icons per result (Unicode escapes)
            $jobLines = @()
            foreach ($r in $results) {
                switch ($r.Status) {
                    "PASS" { $icon = "\u2705" }
                    "FAIL" { $icon = "\u274c" }
                    "SKIP" { $icon = "\u23ed" }
                    default { $icon = "\u2b1c" }
                }
                $t = if ($r.Time -gt 0) { " ($($r.Time)s)" } else { "" }
                $jobLines += "${icon} $($r.Name)${t}"
            }
            $jobsText = $jobLines -join "  "

            # Build JSON payload manually (avoids PS 5.1 ConvertTo-Json Unicode issues)
            $ts = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

            # Failure details field (only when failed)
            $failureField = ""
            $locationField = ""
            if ($failed -and $failureDetails) {
                # Escape JSON special chars in failure details
                $escapedDetails = $failureDetails -replace '\\', '\\\\' -replace '"', '\"' -replace "`r`n", '\n' -replace "`n", '\n' -replace "`t", '  '
                $failureField = ",{`"name`":`"\u26a0 Failure Details`",`"value`":`"``````\n${escapedDetails}\n```````",`"inline`":false}"

                # Extract error locations (file:line:col) from raw output
                $locMatches = [regex]::Matches($failureDetails, '(?m)([a-zA-Z0-9_\-\\\/\.]+\.go:\d+(?::\d+)?)')
                $locations = @()
                foreach ($m in $locMatches) {
                    $loc = $m.Value -replace '\\', '/'
                    if ($locations -notcontains $loc) { $locations += $loc }
                }
                if ($locations.Count -gt 0) {
                    # Limit to 10 locations to avoid embed overflow
                    $shownLocs = $locations | Select-Object -First 10
                    $locLines = ($shownLocs | ForEach-Object { "``$_``" }) -join '\n'
                    if ($locations.Count -gt 10) { $locLines += "\n... +$($locations.Count - 10) more" }
                    $locationField = ",{`"name`":`"\ud83d\udccd Error Locations`",`"value`":`"${locLines}`",`"inline`":false}"
                } else {
                    $locationField = ",{`"name`":`"\ud83d\udccd Error Locations`",`"value`":`"\u0e0a\u0e35\u0e49\u0e08\u0e38\u0e14\u0e44\u0e21\u0e48\u0e44\u0e14\u0e49 \u2014 \u0e14\u0e39 Failure Details \u0e14\u0e49\u0e32\u0e19\u0e1a\u0e19`",`"inline`":false}"
                }
            }

            $payload = @"
{"embeds":[{"title":"$titleEmoji $title","color":$color,"fields":[{"name":"Branch","value":"``$branch``","inline":true},{"name":"Commit","value":"``$sha``","inline":true},{"name":"Machine","value":"${username}@${hostname}","inline":true},{"name":"Message","value":"$commitMsg","inline":false},{"name":"Jobs","value":"$jobsText","inline":false},{"name":"Total Time","value":"${totalSec}s","inline":true}${failureField}${locationField}],"footer":{"text":"ANC Portal CI (Local)"},"timestamp":"$ts"}]}
"@
            # Send as UTF-8
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

    # -- Unknown --

    default {
        Write-Host "Unknown command: $Command" -ForegroundColor Red
        Write-Host "Run: .\run.ps1 help" -ForegroundColor Yellow
    }
}
