# 📧 Email Configuration Guide

SereniBase supports two email configuration options:

1. **MailHog (Default)** - Local email testing without sending real emails
2. **Custom SMTP** - Production email using Gmail, SendGrid, or other SMTP providers

---

## Option 1: MailHog (Recommended for Development)

MailHog is a local email testing tool that catches all emails sent by your application without actually delivering them. This is perfect for development and testing.

### Features
- ✅ **Zero Configuration** - Works out of the box
- ✅ **No Real Emails** - All emails are caught locally
- ✅ **Web Interface** - View all sent emails at http://localhost:8025
- ✅ **Safe Testing** - No risk of sending test emails to real users
- ✅ **Free** - No costs or API keys required

### Configuration
When running the setup script, choose option **1** for MailHog:

```
Choose email configuration:
  1. MailHog (Local testing - recommended for development)
  2. Custom SMTP (Gmail, SendGrid, etc.)

Enter choice [1]: 1
```

This will configure:
```env
EMAIL_SMTP_HOST=mailhog
EMAIL_SMTP_PORT=1025
EMAIL_SMTP_USERNAME=
EMAIL_SMTP_PASSWORD=
EMAIL_FROM_EMAIL=test@example.com
```

### Accessing MailHog Web UI
After starting your services with `docker-compose up`, access the MailHog web interface at:

**http://localhost:8025**

All emails sent by your application will appear here instantly.

### Docker Service
The MailHog service is automatically included in `docker-compose.all.yaml`:

```yaml
mailhog:
  image: mailhog/mailhog:latest
  container_name: mailhog
  ports:
    - "1025:1025"  # SMTP server
    - "8025:8025"  # Web UI
  networks:
    - serenibase-network
```

---

## Option 2: Custom SMTP (Production)

Use a real SMTP server for production environments or when you need to send actual emails.

### Supported Providers
- Gmail (with App Password)
- SendGrid
- Amazon SES
- Mailgun
- Any SMTP-compatible service

### Configuration
When running the setup script, choose option **2** for Custom SMTP:

```
Choose email configuration:
  1. MailHog (Local testing - recommended for development)
  2. Custom SMTP (Gmail, SendGrid, etc.)

Enter choice [1]: 2
```

Then provide your SMTP details:
```
SMTP Host [smtp.gmail.com]: smtp.gmail.com
SMTP Port [587]: 587
SMTP Username (email): your-email@gmail.com
SMTP Password (app password): ****************
From Email [your-email@gmail.com]: your-email@gmail.com
```

### Gmail Configuration

#### Step 1: Enable 2-Factor Authentication
1. Go to your Google Account settings
2. Navigate to **Security**
3. Enable **2-Step Verification**

#### Step 2: Generate App Password
1. Go to **Security** → **2-Step Verification** → **App passwords**
2. Select **Mail** and **Other (Custom name)**
3. Enter "SereniBase" as the name
4. Copy the generated 16-character password
5. Use this password in the setup (not your regular Gmail password)

#### Configuration Values
```env
EMAIL_SMTP_HOST=smtp.gmail.com
EMAIL_SMTP_PORT=587
EMAIL_SMTP_USERNAME=your-email@gmail.com
EMAIL_SMTP_PASSWORD=your-app-password-here
EMAIL_FROM_EMAIL=your-email@gmail.com
```

---

## Switching Between Configurations

### From MailHog to Custom SMTP
1. Edit your `.env` file
2. Update the email configuration:
   ```env
   EMAIL_SMTP_HOST=smtp.gmail.com
   EMAIL_SMTP_PORT=587
   EMAIL_SMTP_USERNAME=your-email@gmail.com
   EMAIL_SMTP_PASSWORD=your-app-password
   EMAIL_FROM_EMAIL=your-email@gmail.com
   ```
3. Restart the email service:
   ```bash
   docker-compose restart email-service
   ```

### From Custom SMTP to MailHog
1. Edit your `.env` file
2. Update the email configuration:
   ```env
   EMAIL_SMTP_HOST=mailhog
   EMAIL_SMTP_PORT=1025
   EMAIL_SMTP_USERNAME=
   EMAIL_SMTP_PASSWORD=
   EMAIL_FROM_EMAIL=test@example.com
   ```
3. Restart the email service:
   ```bash
   docker-compose restart email-service
   ```

---

## Environment Variables Reference

| Variable | Description | Example (MailHog) | Example (Gmail) |
|----------|-------------|-------------------|-----------------|
| `EMAIL_SMTP_HOST` | SMTP server hostname | `mailhog` | `smtp.gmail.com` |
| `EMAIL_SMTP_PORT` | SMTP server port | `1025` | `587` |
| `EMAIL_SMTP_USERNAME` | SMTP username/email | *(empty)* | `user@gmail.com` |
| `EMAIL_SMTP_PASSWORD` | SMTP password | *(empty)* | `app-password` |
| `EMAIL_FROM_EMAIL` | Sender email address | `test@example.com` | `user@gmail.com` |

---

## Troubleshooting

### MailHog Issues

**Problem:** MailHog web UI not accessible
- **Solution:** Check if MailHog container is running: `docker ps | grep mailhog`
- **Solution:** Verify port 8025 is not used by another service

**Problem:** Emails not appearing in MailHog
- **Solution:** Check email service logs: `docker logs email-service`
- **Solution:** Verify `EMAIL_SMTP_HOST=mailhog` in `.env`

### Custom SMTP Issues

**Problem:** Authentication failed
- **Solution:** For Gmail, ensure you're using an App Password, not your regular password
- **Solution:** Verify 2FA is enabled on your Gmail account

**Problem:** Connection timeout
- **Solution:** Check firewall settings
- **Solution:** Verify SMTP port is correct (587 for TLS, 465 for SSL)

**Problem:** Emails going to spam
- **Solution:** Configure SPF/DKIM records for your domain
- **Solution:** Use a verified sender email address

---

## Best Practices

### Development
- ✅ Use **MailHog** for all development work
- ✅ Never use real email credentials in development
- ✅ Check MailHog UI regularly to verify email content

### Testing
- ✅ Use **MailHog** for automated tests
- ✅ Verify email templates using MailHog web UI
- ✅ Test with various email scenarios

### Production
- ✅ Use **Custom SMTP** with production credentials
- ✅ Store credentials in secure environment variables
- ✅ Use a dedicated email service (SendGrid, SES) for reliability
- ✅ Monitor email delivery rates
- ✅ Configure proper SPF/DKIM/DMARC records

---

## Additional Resources

- [MailHog Documentation](https://github.com/mailhog/MailHog)
- [Gmail App Passwords](https://support.google.com/accounts/answer/185833)
- [SendGrid Documentation](https://docs.sendgrid.com/)
- [Amazon SES Documentation](https://docs.aws.amazon.com/ses/)
