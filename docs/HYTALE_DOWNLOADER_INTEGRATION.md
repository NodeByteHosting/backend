## Overview

The Hytale Downloader CLI automates downloading and updating Hytale server files. It integrates with NodeByte's OAuth authentication to provide seamless, secure server provisioning in CI/CD pipelines.

**Download:** https://downloader.hytale.com/hytale-downloader.zip

## Integration Points

```
┌──────────────────────────┐
│  Provisioning Pipeline   │
│  (Terraform/Ansible)     │
└────────────┬─────────────┘
             │
             ├─→ 1. Request OAuth token from NodeByte API
             │   (device code flow or refresh token)
             │
             ├─→ 2. Pass token to Downloader CLI
             │   (via environment variable or CLI flag)
             │
             ├─→ 3. CLI authenticates with Hytale
             │   (using token)
             │
             └─→ 4. Downloads latest server version
                 Extracts and validates files
```

## Prerequisites

- NodeByte backend API running (OAuth endpoints accessible)
- OAuth account with valid refresh token (30-day lifetime)
- Hytale Downloader CLI installed
- Bash/PowerShell for automation scripts
- curl or similar for API calls

## Step 1: Obtain OAuth Tokens

### Option A: Device Code Flow (First-Time Setup)

For initial setup on a new server/CI environment:

**Bash Example:**
```bash
#!/bin/bash

# Request device code
DEVICE_RESPONSE=$(curl -s -X POST http://localhost:3000/api/v1/hytale/oauth/device-code)

DEVICE_CODE=$(echo "$DEVICE_RESPONSE" | jq -r '.device_code')
USER_CODE=$(echo "$DEVICE_RESPONSE" | jq -r '.user_code')
VERIFICATION_URI=$(echo "$DEVICE_RESPONSE" | jq -r '.verification_uri')

echo "🔐 Authorize at: $VERIFICATION_URI"
echo "📝 Enter code: $USER_CODE"
echo ""

# Poll for token (with timeout)
TIMEOUT=900  # 15 minutes
INTERVAL=5
ELAPSED=0

while [ $ELAPSED -lt $TIMEOUT ]; do
    TOKEN_RESPONSE=$(curl -s -X POST http://localhost:3000/api/v1/hytale/oauth/token \
        -H "Content-Type: application/json" \
        -d "{\"device_code\": \"$DEVICE_CODE\"}")
    
    # Check if we got access_token
    if echo "$TOKEN_RESPONSE" | jq -e '.access_token' > /dev/null 2>&1; then
        ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.access_token')
        REFRESH_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.refresh_token')
        
        echo "✅ Authorization successful!"
        echo "ACCESS_TOKEN=$ACCESS_TOKEN" >> .env.local
        echo "REFRESH_TOKEN=$REFRESH_TOKEN" >> .env.local
        break
    fi
    
    sleep $INTERVAL
    ELAPSED=$((ELAPSED + INTERVAL))
done

if [ $ELAPSED -ge $TIMEOUT ]; then
    echo "❌ Authorization timeout (15 min exceeded)"
    exit 1
fi
```

**PowerShell Example:**
```powershell
# Request device code
$response = Invoke-RestMethod -Uri "http://localhost:3000/api/v1/hytale/oauth/device-code" `
    -Method Post -Headers @{"Content-Type" = "application/json"}

$deviceCode = $response.device_code
$userCode = $response.user_code
$verificationUri = $response.verification_uri

Write-Host "🔐 Authorize at: $verificationUri"
Write-Host "📝 Enter code: $userCode"
Write-Host ""

# Poll for token (with timeout)
$timeout = 900  # 15 minutes
$interval = 5
$elapsed = 0

while ($elapsed -lt $timeout) {
    try {
        $tokenResponse = Invoke-RestMethod -Uri "http://localhost:3000/api/v1/hytale/oauth/token" `
            -Method Post `
            -Headers @{"Content-Type" = "application/json"} `
            -Body "{`"device_code`": `"$deviceCode`"}"
        
        if ($tokenResponse.access_token) {
            $accessToken = $tokenResponse.access_token
            $refreshToken = $tokenResponse.refresh_token
            
            Write-Host "✅ Authorization successful!"
            Add-Content -Path ".env.local" -Value "ACCESS_TOKEN=$accessToken"
            Add-Content -Path ".env.local" -Value "REFRESH_TOKEN=$refreshToken"
            break
        }
    } catch {
        # 401 AUTHORIZATION_PENDING is expected until user authorizes
    }
    
    Start-Sleep -Seconds $interval
    $elapsed += $interval
}

if ($elapsed -ge $timeout) {
    Write-Host "❌ Authorization timeout (15 min exceeded)"
    exit 1
}
```

### Option B: Token Refresh (Automated CI/CD)

For automated provisioning using stored refresh token:

**Bash Example:**
```bash
#!/bin/bash

# Load stored refresh token
REFRESH_TOKEN=$(cat .env.local | grep REFRESH_TOKEN | cut -d= -f2)

if [ -z "$REFRESH_TOKEN" ]; then
    echo "❌ No refresh token found. Run device code flow first."
    exit 1
fi

# Refresh token (valid for 30 days)
TOKEN_RESPONSE=$(curl -s -X POST http://localhost:3000/api/v1/hytale/oauth/refresh \
    -H "Content-Type: application/json" \
    -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}")

if echo "$TOKEN_RESPONSE" | jq -e '.access_token' > /dev/null 2>&1; then
    NEW_ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.access_token')
    NEW_REFRESH_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.refresh_token')
    
    # Update stored tokens
    sed -i "s/ACCESS_TOKEN=.*/ACCESS_TOKEN=$NEW_ACCESS_TOKEN/" .env.local
    sed -i "s/REFRESH_TOKEN=.*/REFRESH_TOKEN=$NEW_REFRESH_TOKEN/" .env.local
    
    echo "✅ Token refreshed successfully"
    export ACCESS_TOKEN=$NEW_ACCESS_TOKEN
else
    echo "❌ Token refresh failed"
    echo "$TOKEN_RESPONSE" | jq .
    exit 1
fi
```

**PowerShell Example:**
```powershell
# Load stored refresh token
$env_content = Get-Content ".env.local" -Raw
$refreshToken = ($env_content | Select-String "REFRESH_TOKEN=(.+)").Matches[0].Groups[1].Value

if (-not $refreshToken) {
    Write-Host "❌ No refresh token found. Run device code flow first."
    exit 1
}

# Refresh token
try {
    $tokenResponse = Invoke-RestMethod -Uri "http://localhost:3000/api/v1/hytale/oauth/refresh" `
        -Method Post `
        -Headers @{"Content-Type" = "application/json"} `
        -Body "{`"refresh_token`": `"$refreshToken`"}"
    
    $newAccessToken = $tokenResponse.access_token
    $newRefreshToken = $tokenResponse.refresh_token
    
    # Update stored tokens
    (Get-Content ".env.local") -replace "ACCESS_TOKEN=.*", "ACCESS_TOKEN=$newAccessToken" | Set-Content ".env.local"
    (Get-Content ".env.local") -replace "REFRESH_TOKEN=.*", "REFRESH_TOKEN=$newRefreshToken" | Set-Content ".env.local"
    
    Write-Host "✅ Token refreshed successfully"
    $env:ACCESS_TOKEN = $newAccessToken
} catch {
    Write-Host "❌ Token refresh failed"
    exit 1
}
```

## Step 2: Configure Downloader CLI

### Environment Variables

```bash
# Set these before running downloader CLI

export HYTALE_TOKEN="your_access_token_here"
export HYTALE_SERVER_PATH="/opt/hytale-server"
export HYTALE_ENVIRONMENT="production"  # or "staging"
```

### CLI Usage

```bash
# Download latest server version
./hytale-downloader download \
    --token "$HYTALE_TOKEN" \
    --output "$HYTALE_SERVER_PATH" \
    --version latest

# Verify downloaded files
./hytale-downloader verify \
    --path "$HYTALE_SERVER_PATH"

# Extract files
./hytale-downloader extract \
    --input "$HYTALE_SERVER_PATH" \
    --output "$HYTALE_SERVER_PATH/extracted"
```

## Step 3: Integration Examples

### Terraform Provisioning

```hcl
# variables.tf
variable "nodebyte_api_url" {
  default = "http://localhost:3000"
}

variable "refresh_token" {
  sensitive = true
  # Loaded from environment: TF_VAR_refresh_token
}

# main.tf
resource "null_resource" "download_hytale" {
  provisioner "local-exec" {
    command = <<-EOT
      set -e
      
      # Refresh OAuth token
      TOKEN_RESPONSE=$(curl -s -X POST "${var.nodebyte_api_url}/api/v1/hytale/oauth/refresh" \
        -H "Content-Type: application/json" \
        -d '{"refresh_token": "${var.refresh_token}"}')
      
      ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.access_token')
      
      # Download server files
      ./hytale-downloader download \
        --token "$ACCESS_TOKEN" \
        --output "/opt/hytale-server" \
        --version latest
      
      # Verify installation
      if [ -f /opt/hytale-server/server.jar ]; then
        echo "✅ Server files ready"
      else
        echo "❌ Download verification failed"
        exit 1
      fi
    EOT
  }
}

# terraform.tfvars or environment
# TF_VAR_refresh_token="refresh_eyJhbGc..."
```

### Ansible Playbook

```yaml
---
- name: Provision Hytale Server
  hosts: new_servers
  vars:
    nodebyte_api_url: "http://nodebyte.example.com"
    # Vault-encrypted refresh token
  tasks:
    - name: Refresh OAuth token
      uri:
        url: "{{ nodebyte_api_url }}/api/v1/hytale/oauth/refresh"
        method: POST
        body_format: json
        body:
          refresh_token: "{{ refresh_token }}"
      register: token_response
      changed_when: false

    - name: Extract access token
      set_fact:
        access_token: "{{ token_response.json.access_token }}"
        new_refresh_token: "{{ token_response.json.refresh_token }}"

    - name: Update stored refresh token
      copy:
        content: "{{ new_refresh_token }}"
        dest: "/etc/hytale/refresh_token"
        mode: "0600"

    - name: Download Hytale server files
      shell: |
        ./hytale-downloader download \
          --token "{{ access_token }}" \
          --output "/opt/hytale-server" \
          --version latest
      environment:
        PATH: "/usr/local/bin:{{ ansible_env.PATH }}"

    - name: Verify installation
      stat:
        path: "/opt/hytale-server/server.jar"
      register: server_jar
      failed_when: not server_jar.stat.exists
```

### GitHub Actions CI/CD

```yaml
name: Deploy Hytale Server

on:
  schedule:
    - cron: "0 0 * * 0"  # Weekly
  workflow_dispatch:

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Refresh OAuth token
        id: token
        run: |
          RESPONSE=$(curl -s -X POST "${{ secrets.NODEBYTE_API_URL }}/api/v1/hytale/oauth/refresh" \
            -H "Content-Type: application/json" \
            -d "{\"refresh_token\": \"${{ secrets.HYTALE_REFRESH_TOKEN }}\"}")
          
          ACCESS_TOKEN=$(echo "$RESPONSE" | jq -r '.access_token')
          NEW_REFRESH=$(echo "$RESPONSE" | jq -r '.refresh_token')
          
          echo "::set-output name=access_token::$ACCESS_TOKEN"
          echo "::add-mask::$ACCESS_TOKEN"
          
          # Store new refresh token
          echo "new_refresh=$NEW_REFRESH" >> $GITHUB_OUTPUT

      - name: Update refresh token secret
        run: |
          echo "TODO: Update GitHub Actions secret (requires API token)"
          # This step would use GitHub API to update the secret
          # Requires: gh auth token + gh secret set

      - name: Download Hytale server
        run: |
          wget https://downloader.hytale.com/hytale-downloader.zip
          unzip hytale-downloader.zip
          
          ./hytale-downloader download \
            --token "${{ steps.token.outputs.access_token }}" \
            --output "./server" \
            --version latest

      - name: Verify and package
        run: |
          ./hytale-downloader verify --path "./server"
          tar czf hytale-server.tar.gz server/

      - name: Upload to artifact storage
        run: |
          aws s3 cp hytale-server.tar.gz s3://deployments/hytale/
```

### Docker Build Integration

```dockerfile
# Dockerfile
FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    jq \
    unzip

# Download Downloader CLI
RUN wget https://downloader.hytale.com/hytale-downloader.zip && \
    unzip hytale-downloader.zip && \
    chmod +x hytale-downloader

# Build arguments (passed from CI/CD)
ARG HYTALE_TOKEN
ARG HYTALE_VERSION=latest

# Download server files
RUN ./hytale-downloader download \
    --token "$HYTALE_TOKEN" \
    --output "/opt/hytale-server" \
    --version "$HYTALE_VERSION"

# Verify
RUN ./hytale-downloader verify --path "/opt/hytale-server"

# Copy configuration
COPY server.properties /opt/hytale-server/
COPY start.sh /opt/hytale-server/

# Set permissions
RUN chmod +x /opt/hytale-server/start.sh && \
    useradd -m hytale && \
    chown -R hytale:hytale /opt/hytale-server

USER hytale
WORKDIR /opt/hytale-server

EXPOSE 25565

CMD ["/opt/hytale-server/start.sh"]
```

**Build command:**
```bash
# Get fresh access token
TOKEN_RESPONSE=$(curl -s -X POST http://localhost:3000/api/v1/hytale/oauth/refresh \
    -H "Content-Type: application/json" \
    -d "{\"refresh_token\": \"$REFRESH_TOKEN\"}")

ACCESS_TOKEN=$(echo "$TOKEN_RESPONSE" | jq -r '.access_token')

# Build with token
docker build \
    --build-arg HYTALE_TOKEN="$ACCESS_TOKEN" \
    --build-arg HYTALE_VERSION="1.0.2" \
    -t hytale-server:latest .
```

## Step 4: Error Handling & Retries

### Token Refresh Failures

```bash
#!/bin/bash

refresh_token_with_retry() {
    local refresh_token=$1
    local max_attempts=3
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        RESPONSE=$(curl -s -w "\n%{http_code}" -X POST http://localhost:3000/api/v1/hytale/oauth/refresh \
            -H "Content-Type: application/json" \
            -d "{\"refresh_token\": \"$refresh_token\"}")
        
        HTTP_CODE=$(echo "$RESPONSE" | tail -n 1)
        BODY=$(echo "$RESPONSE" | head -n -1)
        
        case $HTTP_CODE in
            200)
                echo "$BODY"
                return 0
                ;;
            401)
                echo "❌ Refresh token expired or invalid" >&2
                echo "   Need to re-run device code flow" >&2
                return 1
                ;;
            429)
                RETRY_AFTER=$(echo "$BODY" | jq -r '.retry_after // 60')
                echo "⏳ Rate limited. Waiting ${RETRY_AFTER}s..." >&2
                sleep $RETRY_AFTER
                attempt=$((attempt + 1))
                ;;
            *)
                echo "❌ HTTP $HTTP_CODE" >&2
                sleep $((2 ** (attempt - 1)))  # Exponential backoff
                attempt=$((attempt + 1))
                ;;
        esac
    done
    
    echo "❌ Token refresh failed after $max_attempts attempts" >&2
    return 1
}

# Usage
if ! TOKEN_JSON=$(refresh_token_with_retry "$REFRESH_TOKEN"); then
    exit 1
fi

ACCESS_TOKEN=$(echo "$TOKEN_JSON" | jq -r '.access_token')
```

### Downloader CLI Failures

```bash
#!/bin/bash

download_with_retry() {
    local token=$1
    local output=$2
    local max_attempts=3
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if ./hytale-downloader download \
            --token "$token" \
            --output "$output" \
            --version latest 2>&1; then
            return 0
        fi
        
        DELAY=$((2 ** (attempt - 1)))
        echo "⏳ Download failed, retrying in ${DELAY}s... (attempt $attempt/$max_attempts)" >&2
        sleep $DELAY
        attempt=$((attempt + 1))
    done
    
    echo "❌ Download failed after $max_attempts attempts" >&2
    return 1
}

# Usage
if ! download_with_retry "$ACCESS_TOKEN" "/opt/hytale-server"; then
    exit 1
fi

# Verify
if ! ./hytale-downloader verify --path "/opt/hytale-server"; then
    echo "❌ Download verification failed" >&2
    exit 1
fi
```

## Troubleshooting

### "Refresh token expired or invalid"
- Refresh tokens valid for 30 days
- Need to re-run device code flow if expired
- Solution: Implement token rotation in CI/CD pipeline

### "Rate limited (429)"
- Exceeded token refresh quota (6/hour)
- Wait for X-RateLimit-Reset time
- Use stored access tokens if available (valid 1 hour)

### "Session limit exceeded (403)"
- Account reached 100 concurrent sessions
- Solution: User needs to upgrade to `sessions.unlimited_servers` entitlement
- Check `account_id` in token response

### "Authorization pending"
- User hasn't entered device code yet
- Implementation should: retry after 5 seconds per RFC 8628
- Check user has navigated to verification_uri

### "Connection refused"
- NodeByte API not running
- Check API URL and firewall
- Verify OAuth endpoints accessible

## Security Best Practices

1. **Token Storage**
   ```bash
   # ✅ Secure storage
   chmod 600 .env.local  # Read/write for owner only
   
   # ❌ Avoid
   # Don't commit tokens to git
   # Don't print tokens in logs
   # Don't pass via command line (visible in ps)
   ```

2. **CI/CD Secrets**
   ```yaml
   # GitHub Actions
   - name: Use token
     env:
       HYTALE_REFRESH_TOKEN: ${{ secrets.HYTALE_REFRESH_TOKEN }}
     run: |
       # Token masked from logs automatically
       ./deploy.sh
   ```

3. **Network Security**
   - Use HTTPS (not HTTP) for all API calls
   - Verify TLS certificates in production
   - Restrict API access via firewall rules

4. **Token Rotation**
   - New refresh token on every refresh
   - Old token automatically invalidated
   - Store new token immediately

5. **Audit Logging**
   - All downloads logged to audit trail
   - Check `GET /admin/audit-logs?account_id=<id>`
   - Review for suspicious activity

## Summary

The Hytale Downloader CLI integration enables:

- ✅ Automated, secure server provisioning
- ✅ Integration into existing CI/CD pipelines
- ✅ Zero-touch updates via scheduled jobs
- ✅ Compliance-audited downloads
- ✅ Multi-stage/production support

Refer to [HYTALE_AUTH_FLOW.md](HYTALE_AUTH_FLOW.md) for customer-facing authentication documentation and [HYTALE_GSP_API.md](HYTALE_GSP_API.md) for complete API reference.

