# SSL Certificate Configuration

## Cloudflare Origin Certificate

SkillHub is using **Cloudflare Origin Certificate** for SSL/TLS encryption.

### Certificate Details

- **Certificate Path:** `/etc/ssl/cloudflare/aithub.space.pem`
- **Private Key Path:** `/etc/ssl/cloudflare/aithub.space.key`
- **Issuer:** Cloudflare
- **Type:** Origin Certificate (15-year validity)

### How It Works

1. **Client → Cloudflare**: Full SSL/TLS encryption (Cloudflare's public certificate)
2. **Cloudflare → Origin Server**: Encrypted with Cloudflare Origin Certificate
3. **End-to-end encryption** maintained throughout the connection

### Cloudflare SSL/TLS Mode

Ensure Cloudflare is set to **Full (strict)** mode:
- Dashboard → SSL/TLS → Overview → Full (strict)

This ensures:
- ✓ Traffic encrypted between client and Cloudflare
- ✓ Traffic encrypted between Cloudflare and origin server
- ✓ Certificate validation on origin server

### Certificate Renewal

Cloudflare Origin Certificates are valid for **15 years** and do not require renewal like Let's Encrypt certificates.

**Expiration:** Check certificate expiration date:
```bash
openssl x509 -in /etc/ssl/cloudflare/aithub.space.pem -noout -dates
```

### Nginx Configuration

Location: `/etc/nginx/sites-available/aithub.space`

```nginx
ssl_certificate /etc/ssl/cloudflare/aithub.space.pem;
ssl_certificate_key /etc/ssl/cloudflare/aithub.space.key;
```

**Note:** OCSP stapling is disabled for Cloudflare Origin Certificates (not supported).

### Security Features

- **TLS 1.2 and 1.3** only
- **Strong cipher suites** (ECDHE-ECDSA, ECDHE-RSA with AES-GCM)
- **HSTS** enabled (max-age=31536000)
- **Security headers** configured

### Testing SSL

```bash
# Test from server
curl -I https://aithub.space/health

# Test SSL configuration
openssl s_client -connect aithub.space:443 -servername aithub.space

# Check certificate details
echo | openssl s_client -connect aithub.space:443 -servername aithub.space 2>/dev/null | openssl x509 -noout -text
```

### Troubleshooting

**Issue:** SSL handshake errors
```bash
# Check certificate files exist
ls -l /etc/ssl/cloudflare/

# Verify certificate and key match
openssl x509 -noout -modulus -in /etc/ssl/cloudflare/aithub.space.pem | openssl md5
openssl rsa -noout -modulus -in /etc/ssl/cloudflare/aithub.space.key | openssl md5
```

**Issue:** Cloudflare shows "Error 526: Invalid SSL certificate"
- Ensure Cloudflare SSL mode is set to "Full (strict)"
- Verify certificate is properly installed on origin server
- Check Nginx configuration and reload

### Advantages of Cloudflare Origin Certificate

✓ **15-year validity** - No frequent renewals  
✓ **Free** - No cost for certificate  
✓ **Easy setup** - No ACME challenges needed  
✓ **Cloudflare integration** - Seamless with Cloudflare CDN  
✓ **Automatic renewal** - Cloudflare manages certificate lifecycle  

### Backup Certificate Files

**Important:** Keep backup copies of certificate files:

```bash
# Create backup
mkdir -p /opt/backups/ssl
cp /etc/ssl/cloudflare/aithub.space.* /opt/backups/ssl/
chmod 600 /opt/backups/ssl/aithub.space.key
```

### Alternative: Let's Encrypt

If you want to use Let's Encrypt instead:

```bash
# Install certbot
apt install -y certbot python3-certbot-nginx

# Temporarily disable Cloudflare proxy (DNS only)
# Then run:
certbot --nginx -d aithub.space -d www.aithub.space

# Re-enable Cloudflare proxy after certificate is issued
```

**Note:** With Cloudflare proxy enabled, Let's Encrypt HTTP-01 challenge will fail. Use DNS-01 challenge or temporarily disable proxy.

---

**Current Status:** ✅ Cloudflare Origin Certificate active and working
**Last Updated:** 2026-04-20 14:05 UTC
