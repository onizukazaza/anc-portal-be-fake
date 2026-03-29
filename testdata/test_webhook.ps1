# =============================================================================
# ทดสอบ POST /v1/webhooks/github
# =============================================================================
# ใช้คำสั่ง:  .\testdata\test_webhook.ps1
# ต้อง start server ก่อน:  .\run.ps1 dev
# =============================================================================

param(
    [string]$BaseURL = "http://localhost:20000"
)

$endpoint = "$BaseURL/v1/webhooks/github"

# ── จำลอง GitHub Push Event Payload ──
$payload = @'
{
  "ref": "refs/heads/develop",
  "before": "abc1234567890abcdef1234567890abcdef123456",
  "after": "def4567890abcdef1234567890abcdef12345678",
  "compare": "https://github.com/anc-portal/anc-portal-be/compare/abc1234...def4567",
  "commits": [
    {
      "id": "def4567890abcdef1234567890abcdef12345678",
      "message": "feat: add webhook discord notification module",
      "timestamp": "2026-03-29T10:30:00+07:00",
      "url": "https://github.com/anc-portal/anc-portal-be/commit/def4567",
      "author": {
        "name": "guitar",
        "email": "guitar@example.com",
        "username": "guitar"
      }
    },
    {
      "id": "aaa1234567890abcdef1234567890abcdef123456",
      "message": "refactor: clean up config loader",
      "timestamp": "2026-03-29T10:25:00+07:00",
      "url": "https://github.com/anc-portal/anc-portal-be/commit/aaa1234",
      "author": {
        "name": "guitar",
        "email": "guitar@example.com",
        "username": "guitar"
      }
    }
  ],
  "head_commit": {
    "id": "def4567890abcdef1234567890abcdef12345678",
    "message": "feat: add webhook discord notification module",
    "timestamp": "2026-03-29T10:30:00+07:00",
    "url": "https://github.com/anc-portal/anc-portal-be/commit/def4567",
    "author": {
      "name": "guitar",
      "email": "guitar@example.com",
      "username": "guitar"
    }
  },
  "pusher": {
    "name": "guitar",
    "email": "guitar@example.com"
  },
  "sender": {
    "login": "guitar",
    "avatar_url": "https://avatars.githubusercontent.com/u/12345678?v=4",
    "html_url": "https://github.com/guitar"
  },
  "repository": {
    "full_name": "anc-portal/anc-portal-be",
    "html_url": "https://github.com/anc-portal/anc-portal-be"
  }
}
'@

Write-Host ""
Write-Host "  Testing GitHub Webhook Endpoint" -ForegroundColor Cyan
Write-Host "  ================================" -ForegroundColor Cyan
Write-Host "  URL: $endpoint" -ForegroundColor DarkGray
Write-Host ""

# ── Test 1: Push event (should process and send to Discord) ──
Write-Host "  [1] POST push event..." -ForegroundColor Yellow -NoNewline
try {
    $response = Invoke-RestMethod -Uri $endpoint `
        -Method Post `
        -ContentType "application/json" `
        -Headers @{ "X-GitHub-Event" = "push" } `
        -Body ([System.Text.Encoding]::UTF8.GetBytes($payload)) `
        -TimeoutSec 10

    Write-Host " OK" -ForegroundColor Green
    Write-Host "      Response: $($response | ConvertTo-Json -Compress)" -ForegroundColor DarkGray
} catch {
    $status = $_.Exception.Response.StatusCode.value__
    Write-Host " FAIL ($status)" -ForegroundColor Red
    Write-Host "      $($_.Exception.Message)" -ForegroundColor DarkGray
}

# ── Test 2: Non-push event (should be ignored) ──
Write-Host "  [2] POST ping event (should be ignored)..." -ForegroundColor Yellow -NoNewline
try {
    $response = Invoke-RestMethod -Uri $endpoint `
        -Method Post `
        -ContentType "application/json" `
        -Headers @{ "X-GitHub-Event" = "ping" } `
        -Body '{"zen":"test"}' `
        -TimeoutSec 10

    Write-Host " OK" -ForegroundColor Green
    Write-Host "      Response: $($response | ConvertTo-Json -Compress)" -ForegroundColor DarkGray
} catch {
    $status = $_.Exception.Response.StatusCode.value__
    Write-Host " FAIL ($status)" -ForegroundColor Red
    Write-Host "      $($_.Exception.Message)" -ForegroundColor DarkGray
}

# ── Test 3: Invalid signature (should return 401 — only if WEBHOOK_GITHUB_SECRET is set) ──
Write-Host "  [3] POST with bad signature..." -ForegroundColor Yellow -NoNewline
try {
    $response = Invoke-RestMethod -Uri $endpoint `
        -Method Post `
        -ContentType "application/json" `
        -Headers @{
            "X-GitHub-Event" = "push"
            "X-Hub-Signature-256" = "sha256=invalid"
        } `
        -Body ([System.Text.Encoding]::UTF8.GetBytes($payload)) `
        -TimeoutSec 10

    # If no secret is configured, this will succeed
    Write-Host " OK (no secret configured)" -ForegroundColor Green
    Write-Host "      Response: $($response | ConvertTo-Json -Compress)" -ForegroundColor DarkGray
} catch {
    $status = $_.Exception.Response.StatusCode.value__
    if ($status -eq 401) {
        Write-Host " OK (401 as expected)" -ForegroundColor Green
    } else {
        Write-Host " FAIL ($status)" -ForegroundColor Red
    }
    Write-Host "      $($_.Exception.Message)" -ForegroundColor DarkGray
}

Write-Host ""
Write-Host "  Done! Check your Discord channel for the notification." -ForegroundColor Cyan
Write-Host ""
