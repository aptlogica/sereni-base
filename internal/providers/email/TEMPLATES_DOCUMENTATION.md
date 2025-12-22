# Email Templates Documentation

## 1. Email Verification OTP

**Subject:** Email Address Verification Required

**Content:**
```
Dear User,

Thank you for registering with Serenibase.

To complete the verification of your email address, please use the One-Time Password (OTP) provided below:

[OTP Code]

This OTP is valid for a period of 5 minutes.

If you did not initiate this request, please disregard this email. No further action is required.

For security reasons, please do not share this OTP with anyone.
```

---

## 2. Password Reset

**Subject:** Password Reset Instructions

**Content:**
```
Dear User,

We received a request to reset the password associated with your Serenibase account.

To proceed, please click the button below:

[Reset Password Button]

Alternatively, you may copy and paste the following link into your web browser:

[Reset Password URL]

This password reset link will expire in 1 hour for security purposes.

If you did not request a password reset, please ignore this email. Your account will remain secure.
```

---

## 3. Platform Invitation

**Subject:** Invitation to Join [Company Name] on Serenibase

**Content:**
```
Dear [First Name],

You have been invited to join the organization [Company Name] on the Serenibase platform.

To accept this invitation and complete your account setup, please click the button below:

[Accept Invitation Button]

Alternatively, you may use the following link:

[Invitation URL]

This invitation link will remain valid for 1 hour.

If you believe this invitation was sent to you in error, you may safely disregard this message.

We look forward to your participation on Serenibase.
```

---

## 4. Added to Workspace

**Subject:** Access Granted to Workspace: [Workspace Name]

**Content:**
```
Dear User,

You have been granted access to the workspace "[Workspace Name]" on Serenibase.

Assigned Access Level: [Access Level]

You may now log in to begin collaborating within this workspace.

If you have any questions regarding your access, please contact your workspace administrator.
```

---

## 5. Removed from Workspace

**Subject:** Workspace Access Revoked: [Workspace Name]

**Content:**
```
Dear User,

Your access to the workspace "[Workspace Name]" on Serenibase has been removed.

If you believe this change was made in error, please reach out to your workspace administrator for further clarification.
```

---

## 6. Invited to Workspace

**Subject:** Invitation to Join Workspace: [Workspace Name]

**Content:**
```
Dear User,

You have been invited to join the workspace "[Workspace Name]" on Serenibase.

Proposed Access Level: [Access Level]

Please log in to your Serenibase account to review and accept this invitation.

If you were not expecting this invitation, no action is required.
```

---

## 7. Workspace Access Updated

**Subject:** Workspace Access Level Updated

**Content:**
```
Dear User,

Your access permissions for the workspace "[Workspace Name]" have been updated.

New Access Level: [Access Level]

These changes take effect immediately.

If you require additional information regarding this update, please contact your workspace administrator.
```

---

## Template Parameters Reference

| Template | Parameters | Description |
|----------|-----------|-------------|
| Email Verification OTP | `otp` | One-time password, valid for 5 minutes |
| Password Reset | `resetLink` | Secure reset URL, valid for 1 hour |
| Platform Invitation | `firstName`, `companyName`, `resetLink` | Invitation link, valid for 1 hour |
| Added to Workspace | `workspaceName`, `access` | Access confirmation |
| Removed from Workspace | `workspaceName` | Access revocation notice |
| Invited to Workspace | `workspaceName`, `access` | Pending invitation |
| Workspace Access Updated | `workspaceName`, `access` | Access modification notice |

---

## Email Footer

All emails include the following security and confidentiality footer:

```
────────────────────────────────────
CONFIDENTIALITY & SECURITY NOTICE

This email and any attachments are intended solely for the designated recipient and may contain confidential or proprietary information. Unauthorized use, disclosure, or distribution is strictly prohibited.

Serenibase will never request passwords or sensitive credentials via email. Please do not share OTPs, access links, or authentication details.

This is an automated message. Replies are not monitored.

© Serenibase. All rights reserved.
────────────────────────────────────
```

**Note:** The footer is automatically added to all emails via the `constant.EmailFooterNotice` constant defined in `internal/constant/constant.go`.
