# Hytale Authentication for Server Hosting

**Welcome!** This guide explains how Hytale authentication works on NodeByte-hosted servers and how to set up your gaming environment.

## Overview

When you rent a Hytale server from NodeByte, we use secure OAuth 2.0 authentication (the same standard used by Google, Microsoft, and Apple). This means:

- ✅ Your Hytale account stays secure
- ✅ No need to share passwords with us
- ✅ Easy multi-device access
- ✅ Automatic token refresh (you stay logged in)

## How It Works (Simple Version)

```
1. You authorize NodeByte once via web browser
        ↓
2. We get permission to access your Hytale profile
        ↓
3. You select which character to play on the server
        ↓
4. You get a session token to join the game
        ↓
5. Session automatically stays fresh (no re-login needed)
```

**No credentials are ever shared.** Hytale verifies your identity directly.

---

## Authentication Flows

### Flow 1: Interactive Servers (Web Browser)

**Use case:** Setting up a new server, managing multiple accounts

**Step-by-step:**

#### 1️⃣ Go to Your Server Dashboard
Visit your NodeByte control panel and navigate to your server settings.

#### 2️⃣ Click "Connect Hytale Account"
You'll see a blue button to authorize access.

#### 3️⃣ You'll See a Device Code
```
📱 Device Code: AB12-CD34
Please visit: https://accounts.hytale.com/device
and enter the code above
```

#### 4️⃣ Visit the Link in Your Browser
Open `https://accounts.hytale.com/device` in your browser (on any device).

**Note:** You can do this on your phone while your PC is running the server!

#### 5️⃣ Sign In to Hytale (If Not Already)
Enter your Hytale username/password if prompted.

#### 6️⃣ Enter the Device Code
Paste `AB12-CD34` into the code field.

#### 7️⃣ Click "Authorize"
Review permissions:
- ✅ Access your game profiles
- ✅ Create gaming sessions
- ✅ Check your account details

Click the green "Authorize" button.

#### 8️⃣ Done! ✅
Your server is now connected to your Hytale account. Return to your dashboard.

---

### Flow 2: Desktop Applications & Launchers

**Use case:** Dedicated servers, Linux/Mac hosting, automated provisioning

If using a desktop client or server launcher, it might use PKCE (Proof Key for Public Clients) flow:

#### 1️⃣ Launch Your Game/Server Manager
Open the application that needs Hytale access.

#### 2️⃣ Click "Login with Hytale"
The app will open your browser automatically.

#### 3️⃣ Sign In & Authorize
Same as Flow 1 (device code flow), or the app may use a password flow instead.

#### 4️⃣ Redirected Back to App
Once authorized, you're automatically logged in.

---

### Flow 3: Automated/Headless Servers

**Use case:** Dedicated Linux servers, 24/7 game servers, Cloud VPS

For servers without a display/browser:

#### 1️⃣ Get Initial Token (Device Code)
```bash
# Admin runs this command
curl -X POST http://your-server:3000/api/v1/hytale/oauth/device-code

# Output:
# {
#   "user_code": "AB12-CD34",
#   "verification_uri": "https://accounts.hytale.com/device"
# }
```

#### 2️⃣ Authorize from Any Browser
Visit the `verification_uri` from any device and enter the code.

#### 3️⃣ Server Gets Token Automatically
Once authorized, the server receives your access token.

#### 4️⃣ Admin Stores Token Securely
```bash
# Server stores in encrypted config
HYTALE_REFRESH_TOKEN="refresh_eyJhbGc..."

# Token automatically refreshes every 5-10 minutes
# No further action needed
```

---

## Token Lifecycle

### Initial Token (Valid for 1 Hour)

When you first authorize:
```
Time: 0:00      You authorize
      ↓
      0:00      Server gets access token (1 hour expiry)
      ↓
      0:50      Server automatically refreshes token (before expiry)
      ↓
      1:00      Token expired (but we refreshed already!)
      ↓
      1:40      Another refresh happens
      ↓
      Continues forever... you never have to re-auth!
```

### Refresh Token (Valid for 30 Days)

The "refresh token" lets us extend your access:
- ✅ Stored securely on our servers
- ✅ Valid for 30 days
- ✅ Only used to get new access tokens
- ✅ Automatically rotated (new one each refresh)

**Important:** If you don't log in for 30+ days, you'll need to re-authorize.

---

## Join Your Server

### Step 1: Launch Your Game

Open the Hytale launcher (or your server's game client).

### Step 2: Select Server

From the server list, find your NodeByte-hosted server.

### Step 3: Choose Your Character

You'll see a list of all your Hytale characters:
```
Select a character:
○ MyMainCharacter
○ MySecondCharacter
○ MyPvPCharacter
```

Pick the one you want to play with.

### Step 4: Join Game 🎮

Click "Connect" and you'll instantly join the server!

**Behind the scenes:**
1. Your character is verified against your Hytale account ✅
2. Session tokens are generated (valid for 1 hour)
3. Session automatically refreshes every 10 minutes
4. You stay connected as long as you're playing

---

## Multi-Account / Multi-Device Support

### Using Multiple Characters

You can connect any of your Hytale characters to the same server:

1. Open server settings → "Authorize Profile"
2. Select different character from the list
3. Each character keeps their own game progress

### Playing from Different Devices

Your authorization works across devices:

**Device 1 (Desktop):** Your main gaming PC
**Device 2 (Laptop):** While traveling
**Device 3 (Streaming):** Broadcasting on Twitch

Each device can use the same account without re-authorizing!

**How?** The refresh token works anywhere. Your session tokens are created per-device but share the same account credentials.

---

## Troubleshooting

### "Authorization Pending" (Page Won't Update)

**Problem:** You entered the code but the page keeps saying "waiting for authorization"

**Solution:**
1. Check you're on the correct device (where you entered the code)
2. Wait 5-10 seconds - it polls automatically
3. Refresh the original page manually if it's taking too long
4. If it times out after 15 minutes, get a new device code and try again

---

### "Session Limit Exceeded" (403 Error)

**Problem:** You see "This account has reached its concurrent session limit (100 sessions)"

**This means:**
- Your Hytale account has too many active game sessions
- Default limit is 100 concurrent sessions
- Only applies to accounts without premium entitlement

**Solution:**
1. **Terminate old sessions:**
   - Log out from other servers
   - Close other game windows
   - Wait 1 hour for sessions to auto-expire

2. **Upgrade your account:**
   - Purchase the `sessions.unlimited_servers` entitlement
   - After upgrade, no session limit applies
   - Contact Hytale support for details

---

### "Invalid or Expired Token" (401 Error)

**Problem:** You see "Unauthorized - your token expired or is invalid"

**Possible causes:**

1. **30-day refresh token expired**
   - You haven't logged in for 30+ days
   - Solution: Re-authorize once (start with device code flow)

2. **Your password changed**
   - You changed your Hytale password
   - Old tokens automatically invalidated (security feature)
   - Solution: Re-authorize

3. **Account compromised**
   - Someone else may have accessed your account
   - All tokens automatically revoked for safety
   - Solution: Change password at accounts.hytale.com, then re-authorize here

**Re-authorize in 2 steps:**
1. Go to server settings → "Reconnect Hytale Account"
2. Follow the device code flow (same as initial setup)

---

### "Profile Not Found" (404 Error)

**Problem:** "We can't find that character"

**Possible causes:**

1. **Character was deleted**
   - Solution: Create a new character and select it

2. **Using wrong account**
   - You have multiple Hytale accounts
   - Make sure you're signing in with the right one
   - Solution: Sign in with different account at verification_uri

3. **Rare account sync issue**
   - Wait a few minutes and try again
   - Contact NodeByte support if it persists

---

### "The Server Isn't Responding"

**Problem:** "Can't reach the game server" or "Connection refused"

**This is likely a server issue, not auth. Check:**

1. ✅ Is the server running?
   - Visit your NodeByte control panel
   - Check server status (should be green)
   - Click "Start Server" if it's offline

2. ✅ Is the server publicly accessible?
   - Check firewall rules
   - Verify port 25565 (or custom port) is open
   - Check network connectivity

3. ✅ Are you on the right server?
   - Verify the server IP/name
   - Check you're not behind a restrictive firewall/VPN

**If you're still stuck:** Contact NodeByte support with your server ID.

---

### "Rate Limited" (429 Error)

**Problem:** "Too many requests - please slow down"

**This means:**
- You're polling for tokens too frequently
- Default limits: 10 token polls per 5 minutes
- 6 token refreshes per hour

**Solution:**
- Wait a few minutes
- Retry after the `X-RateLimit-Reset` time shown in error
- If using automation, implement exponential backoff

**Don't worry:** Normal gameplay never hits these limits. Only happens with:
- Rapid script/API calls
- Aggressive automated testing

---

## Security & Privacy

### What NodeByte Stores

When you authorize, we store (encrypted):

✅ **Your access token** - Used to create game sessions
✅ **Refresh token** - Used to keep access valid for 30 days
✅ **Account ID** - Links to your Hytale account
✅ **Session history** - For compliance and debugging

### What We Don't Store

❌ Your Hytale password (we never see it)
❌ Your email address (Hytale keeps that)
❌ Your character location/progress (that's in-game data)

### What Hytale Stores

Hytale (the game studio) stores:
- Your character list
- Game progress
- Playtime statistics
- Server access logs

This is standard for any online game.

### How Tokens Are Protected

1. **In Transit:** HTTPS encryption (locked padlock in browser)
2. **At Rest:** AES-256 encryption (military-grade)
3. **Access Control:** Only backend services can decrypt
4. **Audit Logging:** Every token operation is logged

### Revoking Access

Want to disconnect your Hytale account?

1. Go to server settings
2. Click "Disconnect Hytale Account"
3. All tokens for this server are immediately deleted
4. You'll need to re-authorize to play again

**Note:** This only affects this specific server. Your account on other servers continues to work.

---

## Advanced: Manual Token Management

### For Server Admins / Power Users

If you're self-hosting or need manual control:

#### Get a New Token

```bash
# 1. Request device code
curl -X POST http://your-server:3000/api/v1/hytale/oauth/device-code

# Response:
# {
#   "device_code": "DE1234567890ABCDEF",
#   "user_code": "XY12-AB34",
#   "verification_uri": "https://accounts.hytale.com/device",
#   "expires_in": 1800
# }
```

#### Authorize

Visit the `verification_uri` and enter the device code.

#### Retrieve Tokens

```bash
# After authorizing
curl -X POST http://your-server:3000/api/v1/hytale/oauth/token \
  -H "Content-Type: application/json" \
  -d '{"device_code": "DE1234567890ABCDEF"}'

# Response:
# {
#   "access_token": "eyJhbGc...",
#   "refresh_token": "refresh_eyJhbGc...",
#   "expires_in": 3600
# }
```

#### Refresh Token Manually

```bash
curl -X POST http://your-server:3000/api/v1/hytale/oauth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "refresh_eyJhbGc..."}'
```

#### Create Game Session

```bash
curl -X POST http://your-server:3000/api/v1/hytale/oauth/game-session/new \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"profile_uuid": "f47ac10b-58cc-4372-a567-0e02b2c3d479"}'
```

**These endpoints require API key for security. Contact your server admin.**

---

## Glossary

| Term | Explanation |
|------|-------------|
| **OAuth** | Secure authentication standard (used by Google, Microsoft, Apple) |
| **Device Code** | A temporary code (XX00-XX00) you enter to authorize |
| **Access Token** | A password-like credential that's valid for 1 hour |
| **Refresh Token** | A longer-lived credential (30 days) used to get new access tokens |
| **Session Token** | A game session identifier valid for 1 hour, auto-refreshed |
| **Entitlement** | A special account feature/permission (e.g., unlimited_servers) |

---

## Getting Help

### Common Issues & Support

**Issue:** Something isn't working
**Solution:**
1. Check this guide first (especially troubleshooting section)
2. Restart your game client
3. Restart your server
4. Clear browser cookies/cache

**Still stuck?** Contact NodeByte support:
- **Email:** support@nodebyte.com
- **Discord:** https://discord.gg/nodebyte
- **Web:** https://nodebyte.com/support

**Include in your support request:**
- Your server ID
- The error message (screenshot helps!)
- When the issue started
- What you were doing when it happened

### Report a Security Issue

If you find a vulnerability:
- **DO NOT** post publicly
- Email: security@nodebyte.com
- We'll respond within 24 hours

---

## FAQ

**Q: Do I need to re-authorize every time I play?**
A: No! Authorization works forever (until tokens expire after 30 days of inactivity, then you do it once more).

**Q: Can my friend play on my server with their account?**
A: Yes! Each friend authorizes once, then can play any time. Each account is separate.

**Q: What if I lose access to my Hytale account?**
A: Change your password at accounts.hytale.com, then re-authorize here. Old tokens are automatically revoked for security.

**Q: Why do you need permission to access my profile?**
A: To verify which character you're playing and link it to your game progress on the server.

**Q: Is my password ever stored?**
A: No, never. You sign in directly with Hytale. We only get tokens, never passwords.

**Q: Can I see my authorization history?**
A: Yes! Server admins can view audit logs in the control panel (shows all auth events, timestamps, IPs).

**Q: What happens when the game server shuts down?**
A: Tokens remain valid for 30 days. When server restarts, it automatically refreshes tokens.

**Q: Can I use the same token on multiple servers?**
A: Yes, but each server manages its own tokens. For safety, each server gets separate tokens.

---

## Next Steps

1. ✅ **Authorize your account** using the device code flow
2. ✅ **Select your character** from the profile list
3. ✅ **Launch the game** and start playing!
4. ✅ **Your session will auto-refresh** - no action needed

**Questions?** See the troubleshooting section or contact support.

**Happy gaming!** 🎮

---

**Version:** 1.0.0  
**Last Updated:** January 14, 2026  
**Platform:** NodeByte Hytale Hosting

