# API Platform — What We Will Do

A plain-language summary of how we plan to deliver each task in the API Platform roadmap.

## Overview

**What this feature is:** The API Platform lets our customers use SereniBase as the backend for their own websites, mobile apps, and scripts. Each workspace gets its own API address and key, so their apps can store and read data in SereniBase and let their own users sign up and log in, similar to how Supabase works. Workspaces, bases, and tables are still set up in the SereniBase app. The API is for working with the data inside them.

**What we will do in each phase:**

- **Phase 1 — Core APIs & Security:** Give each workspace its own API for adding, viewing, editing, and deleting records. It will include built-in log-in for app users, simple access rules, and API keys. We will also release ready-made toolkits for JavaScript/TypeScript, Go, and Python.
- **Phase 2 — Advanced APIs & Customization:** Add custom actions, more powerful search and filtering, finer control over who can see what, and actions that work on data across a whole workspace. We will then update all three toolkits and publish full guides and examples.

---

**Quick terms**

- **API**: a way for other software to talk to SereniBase and use its data and features.
- **SDK**: a ready-made toolkit that makes our APIs easy to use from a specific programming language.
- **Workspace / Base / Table**: a workspace holds bases (projects), and each base holds tables of data.
- **SereniBase users vs app users**: SereniBase users are our customers' team members, who log in to SereniBase to build and manage a workspace. App users are the people using the apps our customers build. They sign in to that app only, never to SereniBase.
- **Supabase**: a well-known product similar to ours. We will follow its approach, where each project gets its own API address and toolkit setup.

---

## Phase 1 — Core APIs & Security

**Goal:** Give developers a safe, reliable way to connect their apps to SereniBase.

| Task | What we will do | Subtasks |
|---|---|---|
| Workspace, Base & Table APIs + Basic CRUD APIs | Give every workspace its own API address and key (like Supabase), which apps use only to add, view, edit, and delete records in the workspace's tables, following a standard format we document (for example: Acme workspace → Sales base → Customers table). Creating or changing workspaces, bases, and tables is done only in the SereniBase app, based on team roles. | • Give each workspace its own API address<br>• Agree on one standard address format (workspace → base → table)<br>• Show the workspace's API address on a settings page in the app<br>• Add, view, edit, and delete records, one at a time or many at once<br>• Tables and columns created in the SereniBase app become available in the API automatically, with no extra setup<br>• Keep creating and changing workspaces, bases, and tables inside the SereniBase app only, controlled by team roles<br>• Use the same clear success and error messages for every action<br>• Publish an API guide that explains the standard format<br>• Test every action end to end |
| Authentication + Authorization + Authentication APIs | Each workspace keeps its own separate list of app users, the way each Supabase project has its own users, and its API gives apps ready-made sign-up, log-in, log-out, and stay-signed-in. Every request is checked against simple access rules the workspace owner sets (for example, signed-in users can view, only admins can edit), so app users only see and change what they are allowed to. | • Keep a separate app-user list for each workspace<br>• Keep SereniBase team logins and roles (owner, editor, viewer) separate from app users<br>• Sign-up with email and password<br>• Email address confirmation<br>• Log-in and log-out<br>• Keep users signed in securely, and sign them out after a period of inactivity<br>• Forgot password and reset password<br>• Log in with Google, GitHub, and similar accounts (optional)<br>• Basic roles for app users (for example, admin and member)<br>• Simple per-table rules for who can view, add, edit, or delete<br>• Check every request against the key, the user, and the rules<br>• Block repeated failed log-in attempts<br>• Screen in SereniBase to view, block, and remove app users<br>• Record sign-in activity for security checks |
| API Key Management | Let SereniBase users create secret keys for their workspace so their apps and scripts can connect without a person logging in. Users can name, review, and switch off keys at any time if one is lost or no longer needed. | • Create a named key for a workspace<br>• Two key types: a public key that is safe to use in websites and mobile apps (it can only do what the app-user rules allow), and a secret admin key for servers only (full access to the workspace's data)<br>• Show a secret key only once when it is created, and store it securely<br>• List keys with their name, creation date, and last-used date<br>• Switch off a key immediately<br>• Replace a key with a new one without breaking the app<br>• Set an expiry date on a key (optional)<br>• Record how each key is used |
| JavaScript / TypeScript SDK | Build a toolkit for web and Node.js developers. They connect once with their workspace address and key, then add, view, edit, and delete records in any table in a few lines, just like the Supabase toolkit. | • Connect using the workspace address and key<br>• Add, view, edit, and delete records in any table<br>• Sign-up, log-in, and log-out for app users, keeping them signed in automatically<br>• Clear error messages<br>• Typing hints in the code editor for TypeScript users<br>• Works in web browsers and on servers<br>• Automated tests<br>• Publish it so developers can install it with one command (npm)<br>• Version numbers and release notes<br>• Short getting-started guide |
| Go SDK | Build the same toolkit for Go, which is popular for backend services. It connects the same way: workspace address and key first, then the records in any table. | • Connect using the workspace address and key<br>• Add, view, edit, and delete records in any table<br>• Sign-up, log-in, and log-out for app users, and check a signed-in user on the server<br>• Clear error messages<br>• Automated tests<br>• Publish it as a Go package<br>• Version numbers and release notes<br>• Short getting-started guide |
| Python SDK | Build the same toolkit for Python, which is popular for scripts, data work, and automation. It also connects with the workspace address and key. | • Connect using the workspace address and key<br>• Add, view, edit, and delete records in any table<br>• Sign-up, log-in, and log-out for app users<br>• Clear error messages<br>• Works in simple scripts and in high-traffic apps<br>• Automated tests<br>• Publish it so developers can install it with one command (pip)<br>• Version numbers and release notes<br>• Short getting-started guide |

**End of Phase 1:** Developers can securely connect their apps to SereniBase and manage their data using any of three popular languages.

---

## Phase 2 — Advanced APIs & Customization

**Goal:** Make the platform flexible enough for larger and more specialized business needs.

| Task | What we will do | Subtasks |
|---|---|---|
| Custom Endpoints | Let developers add their own API actions for needs that are specific to their business, such as "approve an order" or "generate a monthly summary". | • A way to define a custom action: its name, what information it needs, and what it does<br>• Give each custom action its own address under the workspace's API<br>• Choose who can run it (app users, roles, or keys)<br>• Check the information sent in before the action runs<br>• Test an action inside SereniBase before making it live<br>• Turn actions on and off, edit them, and delete them<br>• Time and usage limits so one action cannot slow down the system<br>• Keep a log of each run and any errors |
| Advanced Queries | Let apps ask more detailed questions of the data: filter, sort, search, view results page by page, and pull related data from linked tables in one request. | • Filter with several conditions (for example, "status is open AND amount over 100")<br>• Sort by one or more columns<br>• Show results page by page<br>• Text search across records<br>• Return only the columns that are needed<br>• Get linked records from related tables in one request<br>• Totals, counts, and averages<br>• Keep common searches fast as data grows<br>• Stop very heavy queries from slowing everyone down |
| Fine-Grained API Control | Give admins more precise control over who can see or change specific tables, records, or fields, and set limits on how much each user or key can use the API. | • Record-level rules (for example, users see only their own records)<br>• Field-level rules (for example, hide salary from non-managers)<br>• Limit a key to read-only or to chosen bases<br>• Usage limits per user and per key<br>• Accept API calls only from approved websites<br>• Accept API calls only from approved locations or networks (optional)<br>• Activity log of who did what, and when<br>• A screen to manage all these rules without writing code |
| Workspace-Level APIs | Let apps work with data across the whole workspace in one go, instead of one base at a time. Managing the workspace itself (members, settings, bases, and tables) stays in the SereniBase app. | • Search for records across all bases in the workspace<br>• Read or update records from several bases in one request<br>• Save related changes across bases together, so either all succeed or none do<br>• Bulk import and export of data for the whole workspace<br>• Usage reports for the workspace (API calls, storage, app users) |
| SDK Enhancements | Update all three toolkits (JavaScript/TypeScript, Go, and Python) so developers can use custom endpoints and advanced queries directly from them. | • Add custom actions to all three toolkits<br>• Add advanced queries to all three toolkits<br>• Add workspace-level actions to all three toolkits<br>• Keep all three toolkits working the same way<br>• Make sure apps built on older versions keep working<br>• Automated tests, new releases, and release notes |
| SDK Documentation & Examples | Write clear setup guides, ready-to-copy examples, and a full reference for each toolkit so developers can start quickly without needing our help. | • Getting-started guide for each toolkit<br>• Full reference of every action<br>• Ready-to-copy examples for common tasks<br>• Small sample apps in each language (for example, a to-do app)<br>• A guide to log-in and access rules<br>• Troubleshooting and FAQ page<br>• Update the docs with every release |

**End of Phase 2:** The platform is customizable, more powerful, and ready for advanced real-world applications.

---

## Overall Outcome

- **Phase 1** gives us a secure, working API platform that developers can use from three languages.
- **Phase 2** makes it flexible and customizable, so it can support bigger and more complex applications.
