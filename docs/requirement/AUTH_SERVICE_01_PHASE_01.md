**🔐 Auth Service**

Rank #01 · Phase 1 - Foundation · 15 Requirements · 11 Independent · 4 Dependent

## **Service Overview**

The Auth Service is the foundational service. Every other service depends on it for user identity, JWT validation, RBAC enforcement, and multi-tenant context injection.

It manages the full user lifecycle: registration → email verification → login (password/OAuth2/LDAP) → JWT/refresh token lifecycle → RBAC role and permission management → multi-tenant isolation.

On the frontend, Auth provides the login/register pages, a user management admin panel, a role & permission matrix editor, and the global auth store (Zustand) that holds the current user context across all pages.

## **Tech Stack**

| **Layer**         | **Technology / Package**          | **Purpose**                                     |
| ----------------- | --------------------------------- | ----------------------------------------------- |
| UI Framework      | React 18 + TypeScript             | Type-safe component tree                        |
| Bundler           | Vite 5                            | Fast HMR and build                              |
| Routing           | React Router v6                   | SPA navigation and nested routes                |
| Server State      | TanStack Query v5                 | API cache, background refetch, mutations        |
| Client State      | Zustand                           | Global UI state (sidebar, user prefs)           |
| Component Library | shadcn/ui (Radix UI primitives)   | Accessible, unstyled - ALL components from here |
| Forms             | React Hook Form + Zod             | Validation schema, field-level errors           |
| Data Tables       | TanStack Table v8                 | Sort, filter, paginate, row selection           |
| Styling           | Tailwind CSS v3                   | Utility-first, no custom CSS                    |
| Icons             | Lucide React                      | Consistent icon set                             |
| HTTP Client       | TanStack Query (fetch under hood) | No raw fetch/axios in components                |
| Toasts            | shadcn Toaster + useToast         | Success/error/info notifications                |
| Date Picker       | shadcn Calendar + Popover         | Date/time inputs                                |
| Code Editor       | Monaco Editor (for SQL)           | SQL Lab and expression editors                  |
| Backend Framework | Gin (Go)                          | HTTP router and middleware                      |
| ORM               | GORM + pgx                        | PostgreSQL access                               |
| Auth              | golang-jwt/jwt RS256              | Token signing & validation                      |
| Password          | bcrypt (cost=12)                  | Credential hashing                              |
| Cache             | go-redis                          | RBAC cache, token blacklist, rate limit         |
| OAuth2            | golang.org/x/oauth2               | Google, Okta, GitHub federation                 |
| LDAP              | go-ldap/ldap v3                   | Enterprise directory auth                       |
| Migrations        | golang-migrate                    | DB schema versioning                            |

| **Attribute**      | **Detail**                                                                                                                 |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| Service Name       | Auth Service                                                                                                               |
| Rank / Build Order | #01                                                                                                                        |
| Phase              | Phase 1 - Foundation                                                                                                       |
| Backend API Prefix | /api/v1/auth · /api/v1/users · /api/v1/roles                                                                               |
| Frontend Routes    | /login · /register · /auth/verify · /settings/users · /settings/roles · /settings/permissions                              |
| Primary DB Tables  | ab_user, ab_role, ab_user_role, ab_permission, ab_view_menu, ab_permission_view, ab_permission_view_role, ab_register_user |
| Total Requirements | 15                                                                                                                         |
| Independent        | 11                                                                                                                         |
| Dependent          | 4                                                                                                                          |

## **Frontend Stack Notes**

Frontend stack mirrors Apache Superset: React 18 + TypeScript, Vite bundler, TanStack Query (React Query) v5 for all server state and API calls, Zustand for global client state, React Router v6 for routing.

Component library: shadcn/ui ONLY - no custom component implementations. Use shadcn primitives: Button, Input, Form, Select, Dialog, Sheet, Table, Tabs, Toast, DropdownMenu, Command, Popover, Badge, Card, Skeleton, Alert, AlertDialog, Tooltip, ScrollArea, Separator, Avatar.

Forms: React Hook Form + Zod schema validation. All form fields must use shadcn Form wrapper with FormField, FormItem, FormLabel, FormControl, FormDescription, FormMessage for consistent error display.

Data tables: shadcn DataTable pattern with TanStack Table v8 (column defs, sorting, pagination, row selection). Never build raw HTML tables.

Notifications: shadcn Toaster + useToast hook. Success toasts = green, error toasts = red, info = default. Never use alert() or custom notification systems.

Loading states: shadcn Skeleton for initial loads. Button loading state via disabled + spinner icon (Lucide Loader2 with animate-spin). Never block UI with full-page spinners.

Styling: Tailwind CSS utility classes only. No inline styles, no CSS modules, no styled-components. Use shadcn CSS variables for theming consistency.

Icons: Lucide React exclusively. Match icon semantics: Plus for create, Pencil for edit, Trash2 for delete, RefreshCw for sync, Download for export, Eye for view, Lock for security.

API integration: all server calls via TanStack Query. useQuery for GET, useMutation for POST/PUT/DELETE. Never use fetch or axios directly in components - always through query hooks in /hooks directory.

Error handling: wrap all page-level components with React Error Boundary. API errors surfaced via toast notifications using onError callback in useMutation.

## **Requirements**

**✓ INDEPENDENT (11) - no cross-service calls required**

**AUTH-001** - **User Self-Registration**

| **Dependency**    | **Priority** | **Phase** | **DB Tables**    | **API / Route**            |
| ----------------- | ------------ | --------- | ---------------- | -------------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | ab_register_user | POST /api/v1/auth/register |

| **⚙️ Backend - Description**
- Accept first_name, last_name, email, username, password. Validate uniqueness across ab_user and ab_register_user. Enforce password complexity (≥12 chars, upper, lower, digit, special). Hash with bcrypt cost=12. Generate 64-byte hex registration_hash. Persist to ab_register_user. Send verification email async.
**🔄 Request Flow**
1. Validate fields → uniqueness check → bcrypt hash → generate hash → GORM.Create → go sendEmail()
**⚙️ Go Implementation**
1. bcrypt.GenerateFromPassword([]byte(password),12)
2. hex.EncodeToString(rand.Read(32 bytes)) → RegistrationHash
3. GORM.Create(&ab_register_user{...})
4. go sendVerificationEmail(email,hash) | **✅ Acceptance Criteria**
- POST /api/v1/auth/register → 201 { message:"Verification email sent" }.
- Duplicate email → 409. Weak password → 400. Missing field → 422.
- Verification email received with /auth/verify?hash= link.
**⚠️ Error Responses**
- 400 - Password complexity.
- 409 - Duplicate email/username.
- 422 - Validation. | **🖥️ Frontend Specification**
**📍 Route & Page**
/register
**🧩 shadcn/ui Components**
- Card + CardHeader + CardContent + CardFooter - page shell
- Form + FormField + FormItem + FormLabel + FormControl + FormMessage - all inputs
- Input (type=text) - first_name, last_name, username
- Input (type=email) - email
- Input (type=password) - password + confirm_password
- Button (type=submit, disabled during mutation) - "Create Account"
- Alert (variant=destructive) - server error display
- Separator + text - "Already have an account? Sign in"
**📦 State & Data Fetching**
- useMutation({ mutationFn: api.register }) → handles POST /api/v1/auth/register
- React Hook Form + Zod schema: z.object({ first_name: z.string().min(1), email: z.string().email(), password: z.string().min(12).regex(/[A-Z]/).regex(/[0-9]/).regex(/[^a-zA-Z0-9]/) })
- On success: navigate to /register/success page showing "Check your email"
**✨ UX Behaviors**
- Password strength indicator: shadcn Progress bar 0-100% below password Input, colored red/orange/green.
- Show/hide password toggle: Lucide Eye/EyeOff icon button inside Input suffix.
- Form submit → Button shows Loader2 animate-spin + disabled until mutation settles.
- Field-level errors appear inline via FormMessage below each FormControl.
- After submit, full form disabled to prevent double-submit.
**🛡️ Client-Side Validation**
- Zod: password min 12 chars + uppercase regex + digit regex + special char regex.
- confirmPassword: .refine(data => data.password === data.confirmPassword, { message: "Passwords do not match", path: ["confirmPassword"] })
- username: z.string().min(3).max(64).regex(/^[a-zA-Z0-9_]+$/)
- email: z.string().email("Enter a valid email address")
**♿ Accessibility (a11y)**
- All inputs have htmlFor labels.
- Error messages linked via aria-describedby.
- Form role="form" with aria-label.
**🌐 API Calls (TanStack Query)**
1. useMutation({ mutationFn: (data) => fetch("/api/v1/auth/register",{method:"POST",body:JSON.stringify(data)}).then(r=>r.json()) }) |
| --- | --- | --- |


**AUTH-002** - **Email Verification & Account Activation**

| **Dependency**    | **Priority** | **Phase** | **DB Tables**              | **API / Route**               |
| ----------------- | ------------ | --------- | -------------------------- | ----------------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | ab_register_user → ab_user | GET /api/v1/auth/verify?hash= |

| **⚙️ Backend - Description**
- Verify registration_hash from URL. Check not expired (24h). Atomic TX: create ab_user, delete ab_register_user. Redirect to /login?activated=true.
**🔄 Request Flow**
1. Parse hash → GORM.First → expiry check → TX(create ab_user, delete reg) → redirect
**⚙️ Go Implementation**
1. db.Transaction(func(tx){ tx.Create(&ab_user{...}); tx.Delete(®) })
2. c.Redirect(302,"/login?activated=true") | **✅ Acceptance Criteria**
- Valid hash → ab_user created, redirect to /login?activated=true.
- Expired → 410. Already used → 404.
**⚠️ Error Responses**
- 404 - Invalid/used hash.
- 410 - Expired link. | **🖥️ Frontend Specification**
**📍 Route & Page**
/auth/verify (handles the email link click)
**🧩 shadcn/ui Components**
- Card + CardContent - centered verification status card
- Alert (variant=default&#124;destructive) - success or error state
- Button - "Go to Login" link button
- Skeleton - shown while verifying (async GET in progress)
**📦 State & Data Fetching**
- useQuery({ queryKey:["verify",hash], queryFn: ()=>api.verify(hash), retry:false }) - fires on mount
- Derive UI state from query: isLoading → Skeleton, isSuccess → green Alert, isError → red Alert + error message
**✨ UX Behaviors**
- On page load: immediately fire GET /api/v1/auth/verify?hash=.
- Loading state: shadcn Skeleton card for ~1s while request in flight.
- Success: green Alert with CheckCircle Lucide icon + "Account activated! You can now sign in."
- Error: red Alert with XCircle icon + specific error (expired / already used).
- Success state: "Go to Login" Button auto-redirects after 3s countdown shown in Badge.
**🛡️ Client-Side Validation**
- hash param validated to be 64 hex chars before firing API call → show 400 Alert if malformed.
**♿ Accessibility (a11y)**
- Alert has role="alert" + aria-live="assertive" so screen readers announce result.
**🌐 API Calls (TanStack Query)**
1. useQuery({ queryKey:["email-verify",hash], queryFn: ()=>fetch("/api/v1/auth/verify?hash="+hash).then(r=>{ if(!r.ok) throw r; return r.json() }) }) |
| --- | --- | --- |


**AUTH-003** - **Login with Username / Password**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**         |
| ----------------- | ------------ | --------- | ------------- | ----------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | ab_user       | POST /api/v1/auth/login |

| **⚙️ Backend - Description**
- Validate credentials, check active, enforce rate limit (20/min/IP) and account lockout (5 failures → 15min). Return RS256 JWT (15min) + refresh token (7d, HttpOnly cookie). Update login_count, last_login.
**🔄 Request Flow**
1. Rate limit → find user → lockout check → bcrypt.Compare → update counters → jwt.Sign → set cookie → return tokens
**⚙️ Go Implementation**
1. redis.Incr("rate:login:"+ip) Expire(60s) → 429 if >20
2. bcrypt.CompareHashAndPassword
3. jwt.NewWithClaims(RS256,claims).SignedString(privateKey)
4. redis.Set("refresh:"+token,userID,7*24*time.Hour)
5. c.SetCookie("refresh_token",refresh,7*24*3600,"/","",true,true) | **✅ Acceptance Criteria**
- 200 + {access_token, refresh_token}.
- 5× bad password → 423 Locked.
- Rate limit → 429.
- Inactive → 403.
**⚠️ Error Responses**
- 401 - Bad credentials.
- 403 - Inactive.
- 423 - Locked.
- 429 - Rate limit. | **🖥️ Frontend Specification**
**📍 Route & Page**
/login
**🧩 shadcn/ui Components**
- Card + CardHeader + CardTitle + CardDescription + CardContent + CardFooter - page shell
- Form + FormField + FormItem + FormLabel + FormControl + FormMessage - all inputs
- Input (type=text) - username or email
- Input (type=password) - password with show/hide toggle
- Button (type=submit) - "Sign In" with loading state
- Alert (variant=destructive) - lockout / inactive / server error
- Separator - between form and OAuth options
- Button (variant=outline) × N - one per OAuth provider (Google, etc.) with provider icon
**📦 State & Data Fetching**
- Zustand authStore: { user, accessToken, isAuthenticated, setAuth, clearAuth }
- useMutation({ mutationFn: api.login, onSuccess: (data)=>{ authStore.setAuth(data); navigate("/") }, onError: (err)=>{ toast.error(err.message) } })
- React Hook Form: { username: z.string().min(1), password: z.string().min(1) }
- Persist accessToken in memory (NOT localStorage). Refresh token in HttpOnly cookie.
**✨ UX Behaviors**
- If redirected from protected route: show shadcn Alert (info) "Sign in to continue".
- ?activated=true query param: show success Alert "Account activated! Welcome aboard."
- Submit: Button disabled + Loader2 spinner during login request.
- Lockout error: Alert shows countdown timer until account unlocks (JS setInterval).
- Remember username: shadcn Checkbox "Remember me" → saves username to localStorage only.
- Forgot password link → /forgot-password route.
**🛡️ Client-Side Validation**
- username + password both required (non-empty Zod).
- No client-side length or complexity on login - server handles.
**♿ Accessibility (a11y)**
- autocomplete="username" and autocomplete="current-password" on inputs.
- Button aria-busy=true during submission.
**🌐 API Calls (TanStack Query)**
1. useMutation({ mutationFn: (creds)=>fetch("/api/v1/auth/login",{method:"POST",body:JSON.stringify(creds)}).then(r=>r.json()) }) |
| --- | --- | --- |


**AUTH-004** - **JWT Middleware & Token Validation**

| **Dependency**    | **Priority** | **Phase** | **DB Tables**  | **API / Route**                            |
| ----------------- | ------------ | --------- | -------------- | ------------------------------------------ |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | ab_user (read) | Internal middleware - all protected routes |

| **⚙️ Backend - Description**
- Gin middleware: extract Bearer token → verify RS256 → check jti blacklist → load user from Redis cache → inject UserContext into Gin context. <2ms p99.
**🔄 Request Flow**
1. Extract header → ParseWithClaims → check blacklist → cache lookup → inject ctx → Next()
**⚙️ Go Implementation**
1. jwt.ParseWithClaims(token,&Claims{},keyFunc)
2. redis.Exists("jwt:blacklist:"+jti) → 401 if found
3. redis.Get("user:"+uid) → if miss: GORM.First → cache 5min
4. c.Set("user",UserContext{...}); c.Next() | **✅ Acceptance Criteria**
- Valid token → ctx["user"] set, handler proceeds.
- Missing/expired/tampered/revoked → 401.
- Deactivated user → 403.
**⚠️ Error Responses**
- 401 - All token failures.
- 403 - Inactive user. | **🖥️ Frontend Specification**
**📍 Route & Page**
N/A - frontend-side token management via Zustand + axios/fetch interceptor
**🧩 shadcn/ui Components**
- No UI component - this is an HTTP interceptor concern
- Toaster (shadcn) - surfaced when token expires mid-session
**📦 State & Data Fetching**
- Zustand authStore.accessToken - stored in memory, never localStorage
- TanStack Query: set retry: false for 401 errors, redirect to /login instead
- Axios/fetch interceptor: on 401 response → attempt refresh (POST /auth/refresh) → retry original request → if refresh fails: authStore.clearAuth() + navigate("/login")
- Silent refresh: schedule token refresh at (exp - 60s) using setTimeout stored in authStore
**✨ UX Behaviors**
- Token expiry mid-session: transparent silent refresh (user sees nothing).
- Silent refresh failure: shadcn Toast (variant=destructive) "Your session expired. Please sign in again." → redirect /login.
- Protected routes: React Router  wrapper component checks authStore.isAuthenticated, redirects /login with state={from:currentPath}.
- On login after redirect: navigate to state.from or "/" (saved redirect).
**🛡️ Client-Side Validation**
- Client checks token exp claim before making API calls (avoids unnecessary 401s).
**♿ Accessibility (a11y)**
- Session expiry toast has role="alert" for screen reader announcement.
**🌐 API Calls (TanStack Query)**
1. Interceptor: fetch wrapper adds Authorization: Bearer {accessToken} header
2. Silent refresh: useMutation({ mutationFn: ()=>fetch("/api/v1/auth/refresh",{method:"POST"}) }) |
| --- | --- | --- |


**AUTH-005** - **Refresh Token Rotation**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**           |
| ----------------- | ------------ | --------- | ------------- | ------------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | - Redis only  | POST /api/v1/auth/refresh |

| **⚙️ Backend - Description**
- Validate refresh token from Redis. Rotate: atomically delete old + insert new. Detect reuse attacks (already-deleted = stolen → invalidate all sessions). Issue new access token with re-fetched roles.
**🔄 Request Flow**
1. redis.Get(refresh) → validate → Del old → Set new → re-fetch roles → jwt.Sign → return
**⚙️ Go Implementation**
1. redis.Del("refresh:"+token) → 0 returned = reuse → scan+del all user tokens
2. jwt.NewWithClaims(RS256,updatedRoleClaims).Sign(privateKey) | **✅ Acceptance Criteria**
- 200 + new tokens.
- Reuse of rotated token → 401 + all sessions killed.
- Expired → 401.
**⚠️ Error Responses**
- 401 - Invalid/reuse/expired. | **🖥️ Frontend Specification**
**📍 Route & Page**
N/A - called by interceptor, no UI page
**🧩 shadcn/ui Components**
- No direct UI component
**📦 State & Data Fetching**
- Silent refresh logic in authStore.refresh() action
- On success: authStore.setAccessToken(newToken) + reschedule next refresh
- On failure: authStore.clearAuth() + navigate("/login")
**✨ UX Behaviors**
- 100% transparent to user during normal flow.
- Failure surfaces as session-expired Toast (see AUTH-004).
**🌐 API Calls (TanStack Query)**
1. fetch("/api/v1/auth/refresh",{method:"POST",credentials:"include"}) // sends HttpOnly cookie |
| --- | --- | --- |


**AUTH-006** - **Logout & Token Revocation**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**          |
| ----------------- | ------------ | --------- | ------------- | ------------------------ |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | - Redis only  | POST /api/v1/auth/logout |

| **⚙️ Backend - Description**
- Blacklist access token jti in Redis (remaining TTL). Delete refresh token. Clear HttpOnly cookie. Support logout-all-devices via all=true param.
**🔄 Request Flow**
1. Extract jti → redis.Set(blacklist,ttl) → redis.Del(refresh) → clearCookie → 204
**⚙️ Go Implementation**
1. redis.Set("jwt:blacklist:"+jti,"1",remainingTTL)
2. c.SetCookie("refresh_token","",0,"/","",true,true) | **✅ Acceptance Criteria**
- 204 always.
- Access token rejected after logout.
- logout?all=true kills all sessions.
**⚠️ Error Responses**
- 204 - Always (idempotent). | **🖥️ Frontend Specification**
**📍 Route & Page**
Triggered from /settings or header dropdown - no dedicated page
**🧩 shadcn/ui Components**
- DropdownMenu in TopNav - "Sign out" item with LogOut Lucide icon
- AlertDialog - "Sign out from all devices?" confirmation for logout-all
- DropdownMenuSeparator - visual separation before logout item
**📦 State & Data Fetching**
- useMutation({ mutationFn: api.logout, onSuccess: ()=>{ authStore.clearAuth(); navigate("/login") } })
- authStore.clearAuth() → sets user=null, accessToken=null, isAuthenticated=false
**✨ UX Behaviors**
- TopNav UserAvatar → DropdownMenu: { Profile, Settings, Separator, "Sign out" }.
- "Sign out from all devices" in dropdown → AlertDialog confirmation → POST /logout?all=true.
- After logout: navigate /login, clear TanStack Query cache (queryClient.clear()).
- Loading state on DropdownMenu item: disable item + show Loader2 while request in flight.
**🌐 API Calls (TanStack Query)**
1. useMutation({ mutationFn: (all)=>fetch("/api/v1/auth/logout?all="+all,{method:"POST",credentials:"include"}) }) |
| --- | --- | --- |


**AUTH-007** - **Role CRUD Management**

| **Dependency**    | **Priority** | **Phase** | **DB Tables**         | **API / Route**                                       |
| ----------------- | ------------ | --------- | --------------------- | ----------------------------------------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | ab_role, ab_user_role | GET/POST /api/v1/admin/roles · PUT/DELETE /api/v1/admin/roles/:id |

| **⚙️ Backend - Description**
- Admin CRUD on roles. Guard: cannot delete built-in roles or roles with assigned users. List includes user_count and permission_count. Cache bust after changes.
**🔄 Request Flow**
1. Validate Admin role → GORM CRUD on ab_role → guard checks → cache bust
**⚙️ Go Implementation**
1. GORM.Create(&ab_role{Name:name})
2. GORM.Where("role_id=?",id).Count(&n) → 409 if n>0
3. redis.Del("rbac:*") | **✅ Acceptance Criteria**
- POST → 201.
- DELETE with users → 409.
- Delete built-in → 403.
- GET → list with counts.
**⚠️ Error Responses**
- 403 - Non-admin or built-in role.
- 409 - Role has users. | **🖥️ Frontend Specification**
**📍 Route & Page**
admin/settings/roles
**🧩 shadcn/ui Components**
- DataTable (TanStack Table) - columns: Name, Users, Permissions, Actions
- Button (+ New Role) - opens Dialog
- Dialog + DialogContent + DialogHeader + DialogTitle + DialogFooter - create/edit role modal
- Form + FormField + Input - role name input inside Dialog
- AlertDialog - delete confirmation "Delete role {name}? This cannot be undone."
- Badge - user_count and permission_count display
- DropdownMenu (Actions column) - Edit, Delete items
- Tooltip - "Built-in roles cannot be deleted" on disabled delete for system roles
**📦 State & Data Fetching**
- useQuery({ queryKey:["roles"], queryFn: api.getRoles }) - DataTable data source
- useMutation({ mutationFn: api.createRole, onSuccess: ()=>{ queryClient.invalidateQueries(["roles"]); toast.success("Role created") } })
- useMutation({ mutationFn: api.deleteRole, onError: (e)=>toast.error(e.message) })
- local useState: { isCreateOpen, isDeleteOpen, selectedRole }
**✨ UX Behaviors**
- Table column "Users": Badge with user count, click → opens Sheet with user list.
- Table column "Permissions": Badge with count, click → navigates to role permission matrix.
- Delete: AlertDialog with role name in bold + warning if user_count > 0.
- Create/Edit Dialog: Input auto-focused on open, Enter submits form.
- Toast on success, Toast (destructive) on error.
- Optimistic update: role appears in table immediately on create, removed on delete.
**🛡️ Client-Side Validation**
- Role name: z.string().min(1,"Name is required").max(64,"Max 64 chars")
**♿ Accessibility (a11y)**
- Dialog has aria-labelledby pointing to DialogTitle.
- DataTable rows navigable with keyboard (tabIndex on rows).
**🌐 API Calls (TanStack Query)**
1. useQuery({ queryKey:["roles"], queryFn: ()=>fetch("/api/v1/roles").then(r=>r.json()) })
2. useMutation({ mutationFn: (role)=>fetch("/api/v1/roles",{method:"POST",body:JSON.stringify(role)}) }) |
| --- | --- | --- |


**AUTH-008** - **Permission & View Menu Management**

| **Dependency**    | **Priority** | **Phase** | **DB Tables**                                   | **API / Route**                                                                                       |
| ----------------- | ------------ | --------- | ----------------------------------------------- | ----------------------------------------------------------------------------------------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | ab_permission, ab_view_menu, ab_permission_view | GET/POST /api/v1/permissions · GET/POST /api/v1/admin/view-menus · GET/POST/DELETE /api/v1/admin/permission-views |

| **⚙️ Backend - Description**
- Admin CRUD for permission actions, view menus, and their combinations (permission_views). Permission_views are seeded at startup. Delete guarded by role assignment count.
**🔄 Request Flow**
1. GORM CRUD on ab_permission, ab_view_menu, ab_permission_view with guards
**⚙️ Go Implementation**
1. GORM.FirstOrCreate(&ab_permission,ab_permission{Name:name}) for seed
2. GORM.Where("permission_view_id=?",id).Count on ab_permission_view_role → 409 | **✅ Acceptance Criteria**
- POST permission → 201.
- Duplicate perm_view → 409.
- Delete assigned perm_view → 409.
**⚠️ Error Responses**
- 409 - Duplicate or in-use. | **🖥️ Frontend Specification**
**📍 Route & Page**
admin/settings/permissions
**🧩 shadcn/ui Components**
- Tabs + TabsList + TabsTrigger + TabsContent - "Permissions" &#124; "View Menus" &#124; "Permission Matrix"
- DataTable - list permissions and view menus in their respective tabs
- Command + CommandInput + CommandList + CommandItem - searchable dropdown to create permission_view pairs
- Badge - display permission name and view menu name in matrix cells
- ScrollArea - scrollable permission matrix grid
- Skeleton - loading state for each tab
**📦 State & Data Fetching**
- useQuery({ queryKey:["permissions"] }) - tab 1 data
- useQuery({ queryKey:["view-menus"] }) - tab 2 data
- useQuery({ queryKey:["permission-views"] }) - matrix data
- useMutation for create/delete permission_view pairs
**✨ UX Behaviors**
- Permission Matrix tab: grid with view_menus as columns, permissions as rows, checkboxes at intersections.
- Check a cell → POST /permission-views. Uncheck → DELETE /permission-views/:id.
- Bulk save: "Save Changes" Button collects all diff → single batch API call.
- Command search filters the grid rows/columns in real-time.
**🌐 API Calls (TanStack Query)**
1. useQuery(["permission-views"])
2. useMutation({ mutationFn: (pair)=>fetch("/api/v1/permission-views",{method:"POST",...}) }) |
| --- | --- | --- |


**AUTH-009** - **Assign Permissions to Role (RBAC Matrix)**

| **Dependency**    | **Priority** | **Phase** | **DB Tables**           | **API / Route**                                                                                                          |
| ----------------- | ------------ | --------- | ----------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | ab_permission_view_role | PUT /api/v1/admin/roles/:id/permissions · POST /api/v1/admin/roles/:id/permissions/add · DELETE /api/v1/admin/roles/:id/permissions/:pv_id |

| **⚙️ Backend - Description**
- Replace-all permission assignment to a role. Also additive add and single revoke. After any change, bust RBAC Redis cache for all users with this role.
**🔄 Request Flow**
1. TX: delete existing → bulk insert new → cache bust for affected users
**⚙️ Go Implementation**
1. db.Transaction(func(tx){ tx.Where("role_id=?",id).Delete; tx.CreateInBatches(newRows,100) })
2. redis.Del("rbac:"+userID) for each user with this role | **✅ Acceptance Criteria**
- PUT with [1,2,3] → 200.
- Invalid pv_id → 422.
- Cache busted for affected users.
**⚠️ Error Responses**
- 422 - Invalid permission_view_id.
- 403 - Non-admin. | **🖥️ Frontend Specification**
**📍 Route & Page**
/settings/roles/:id/permissions
**🧩 shadcn/ui Components**
- Card - page container with role name in CardHeader
- ScrollArea - scrollable permission list
- Checkbox - per permission_view row, checked = assigned
- Button ("Save Changes", variant=default) - triggers PUT with all checked IDs
- Button ("Reset", variant=ghost) - reverts local changes to server state
- Badge - category grouping (e.g., "Dataset", "Dashboard", "SQLLab")
- Input + Lucide Search icon - filter permissions by name
- Skeleton - loading state
- Toast - success/error feedback
**📦 State & Data Fetching**
- useQuery({ queryKey:["role-permissions",roleId] }) - current assignments
- useState: localAssignments (Set) - tracks current checkbox state
- useMutation({ mutationFn: (ids)=>api.setRolePermissions(roleId,ids) })
- isDirty: compare localAssignments vs server state → show "unsaved changes" Badge
**✨ UX Behaviors**
- "Save Changes" Button disabled until isDirty=true.
- Unsaved changes indicator: shadcn Badge "Unsaved changes" in page header area.
- Group permissions by view_menu name with Separator between groups.
- Search Input filters visible rows (client-side, no API call).
- Confirm before navigating away with unsaved changes: browser beforeunload + React Router blocker.
**🛡️ Client-Side Validation**
- At least one permission must be assigned - disable Save if Set is empty.
**🌐 API Calls (TanStack Query)**
1. useMutation({ mutationFn: (ids)=>fetch("/api/v1/roles/"+roleId+"/permissions",{method:"PUT",body:JSON.stringify({permission_view_ids:ids})}) }) |
| --- | --- | --- |


**AUTH-010** - **Assign Roles to User**

| **Dependency**    | **Priority** | **Phase** | **DB Tables** | **API / Route**                                           |
| ----------------- | ------------ | --------- | ------------- | --------------------------------------------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | ab_user_role  | PUT /api/v1/admin/users/:id/roles · GET /api/v1/admin/users/:id/roles |

| **⚙️ Backend - Description**
- Replace-all role assignment to a user (Admin only). Must keep at least 1 role. Invalidate RBAC cache for user.
**🔄 Request Flow**
1. Validate Admin → validate ≥1 role → TX delete+insert → redis.Del("rbac:"+uid)
**⚙️ Go Implementation**
1. TX: Delete existing ab_user_role for user; CreateInBatches new
2. redis.Del("rbac:"+userID) | **✅ Acceptance Criteria**
- PUT with [1,3] → 200.
- Empty roles → 422.
- Non-admin → 403.
**⚠️ Error Responses**
- 403 - Non-admin.
- 422 - Empty roles or invalid role_id. | **🖥️ Frontend Specification**
**📍 Route & Page**
/settings/users/:id (user detail page, roles section)
**🧩 shadcn/ui Components**
- Sheet (from table row action) or inline section in user detail page
- Select (multi) - shadcn MultiSelect via Command + Popover pattern for role selection
- Badge × N - display currently assigned roles with X remove button
- Button ("Update Roles") - save changes
- Alert (variant=destructive) - "User must have at least one role" error
**📦 State & Data Fetching**
- useQuery({ queryKey:["user-roles",userId] }) - current roles
- useState: selectedRoleIds (number[]) - local edit state
- useMutation({ mutationFn: (ids)=>api.setUserRoles(userId,ids) })
**✨ UX Behaviors**
- Multi-select: Command + Popover pattern (shadcn standard multi-select). User types to filter roles.
- Selected roles shown as Badge list below select. Click X on Badge removes role.
- Minimum 1 role enforced: Save Button disabled if selectedRoleIds.length===0.
- Toast success on save.
**🛡️ Client-Side Validation**
- selectedRoleIds.length >= 1 - client enforced before API call.
**🌐 API Calls (TanStack Query)**
1. useMutation({ mutationFn: (ids)=>fetch("/api/v1/users/"+userId+"/roles",{method:"PUT",body:JSON.stringify({role_ids:ids})}) }) |
| --- | --- | --- |


**AUTH-011** - **RBAC Permission Check Middleware**

| **Dependency**    | **Priority** | **Phase** | **DB Tables**                                       | **API / Route**                                     |
| ----------------- | ------------ | --------- | --------------------------------------------------- | --------------------------------------------------- |
| **✓ INDEPENDENT** | **P0**       | Phase 1   | ab_user_role, ab_permission_view_role (Redis cache) | Internal middleware - wraps all protected endpoints |

| **⚙️ Backend - Description**
- Gin middleware factory RequirePermission("action","resource"). Resolves user RBAC from Redis cache (TTL 5min). Admin bypass. Rejects with 403 if missing.
**🔄 Request Flow**
1. Get UserContext → Admin bypass → cache check → DB join if miss → evaluate → Next() or 403
**⚙️ Go Implementation**
1. func RequirePermission(perm,view string) gin.HandlerFunc
2. redis.SMembers("rbac:"+uid) → if miss: DB join query → redis.SAdd(...) Expire(5min)
3. if !set.Contains(perm+":"+view): c.AbortWithStatusJSON(403,...) | **✅ Acceptance Criteria**
- Authorized → handler proceeds.
- Unauthorized → 403.
- Cache hit <1ms.
**⚠️ Error Responses**
- 403 - Permission denied. | **🖥️ Frontend Specification**
**📍 Route & Page**
N/A - handled via route guards in React Router
**🧩 shadcn/ui Components**
-  - HOC that renders children or 403 page
- Card (centered) - 403 "Access Denied" page with ShieldAlert Lucide icon
- Button - "Go Back" navigation
**📦 State & Data Fetching**
- authStore.user.permissions (Set) - loaded from JWT claims on login
- hasPermission(perm,resource): checks authStore.user.permissions.has(perm+":"+resource)
- usePermission(perm,resource): hook returning bool, used in components to conditionally render UI
**✨ UX Behaviors**
- Navigation items hidden (not just disabled) when user lacks permission to see them.
- Action buttons (Edit, Delete) hidden via usePermission hook - not just disabled.
- 403 page: shadcn Card centered on screen, ShieldAlert icon, "You don't have permission to access this page." text, "Go back" Button.
**🌐 API Calls (TanStack Query)**
1. No API call - derived from JWT claims in authStore |
| --- | --- | --- |


**⚠ DEPENDENT (4) - requires prior services/requirements**

**AUTH-012** - **Multi-Tenant Context Injection**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**             | **API / Route**                      |
| --------------- | ------------ | --------- | ------------------------- | ------------------------------------ |
| **⚠ DEPENDENT** | **P0**       | Phase 1   | ab_user (org_id from JWT) | Internal middleware - all DB queries |

**⚑ Depends on:** AUTH-004 - JWT middleware must run first to populate user context

| **⚙️ Backend - Description**
- After JWT validation, inject GORM TenantScope(orgID) into all downstream queries. Admin can override with X-Org-Id header for cross-tenant ops.
**🔄 Request Flow**
1. Extract orgID from JWT → if Admin + X-Org-Id: use header → db.Scopes(TenantScope) → c.Set("db",scopedDB) → Next()
**⚙️ Go Implementation**
1. scopedDB:=db.WithContext(ctx).Scopes(TenantScope(orgID))
2. c.Set("db",scopedDB) | **✅ Acceptance Criteria**
- All queries auto-filtered by org_id.
- Cross-tenant access → 404 (not 403).
**⚠️ Error Responses**
- 404 - Cross-tenant resource (hidden, not 403). | **🖥️ Frontend Specification**
**📍 Route & Page**
N/A - org_id is transparent to the frontend
**🧩 shadcn/ui Components**
- OrgSwitcher in TopNav (if user belongs to multiple orgs)
**📦 State & Data Fetching**
- authStore.user.orgId - from JWT.
- If multi-org: OrgSwitcher dropdown → triggers re-login flow with new orgId
**✨ UX Behaviors**
- OrgSwitcher: shadcn Select in TopNav showing current org name. Changing org = new JWT required.
**🌐 API Calls (TanStack Query)**
1. N/A - backend handles all org scoping transparently |
| --- | --- | --- |


**AUTH-013** - **OAuth2 Federation (Google / Okta)**

| **Dependency**  | **Priority** | **Phase** | **DB Tables** | **API / Route**                                                                          |
| --------------- | ------------ | --------- | ------------- | ---------------------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 2   | ab_user       | GET /api/v1/auth/oauth2/:provider/authorize · GET /api/v1/auth/oauth2/:provider/callback |

**⚑ Depends on:** AUTH-003 (user upsert logic), AUTH-010 (default role assignment)

| **⚙️ Backend - Description**
- OAuth2 PKCE flow. State + code_verifier stored in Redis 10min. Exchange code → fetch userinfo → upsert ab_user → issue JWT. Auto-provision controlled by OAUTH_AUTO_PROVISION env.
**🔄 Request Flow**
1. Generate state+verifier → redirect → provider → callback → validate state → exchange → userinfo → upsert → JWT
**⚙️ Go Implementation**
1. golang.org/x/oauth2
2. redis.Set("oauth:state:"+state,provider,10min)
3. GORM.FirstOrCreate(&ab_user,ab_user{Email:email}) | **✅ Acceptance Criteria**
- Valid callback → JWT returned.
- State mismatch → 400.
- New user with provision=true → Gamma role assigned.
**⚠️ Error Responses**
- 400 - State mismatch.
- 403 - Auto-provision disabled.
- 502 - Provider unreachable. | **🖥️ Frontend Specification**
**📍 Route & Page**
/login (OAuth buttons) + /auth/oauth2/callback (redirect target)
**🧩 shadcn/ui Components**
- Button (variant=outline) per provider - "Continue with Google", "Continue with Okta"
- Provider SVG icon inside Button (loaded as React component)
- Separator with "or" text - between password form and OAuth buttons
- Card (full page) on /auth/oauth2/callback - loading state while processing
- Skeleton - shown while /callback page processes the exchange
**📦 State & Data Fetching**
- OAuth redirect: window.location.href = "/api/v1/auth/oauth2/google/authorize" - full page redirect
- /callback page: parse code+state from URL → fire GET /api/v1/auth/oauth2/google/callback (automatic, backend handles)
- Backend redirects /callback to frontend /oauth-success?token=... → authStore.setAuth(token) → navigate("/")
**✨ UX Behaviors**
- OAuth Button: click → full browser redirect to provider (not popup).
- /callback page shown while backend processes: Skeleton card "Signing you in..." + Loader2.
- On success: toast "Welcome back, {name}!" (if returning) or "Account created!" (if new).
- On error: navigate /login?oauth_error= → Alert shown on login page.
**🌐 API Calls (TanStack Query)**
1. window.location.href = "/api/v1/auth/oauth2/"+provider+"/authorize" // full redirect, not fetch |
| --- | --- | --- |


**AUTH-014** - **LDAP Authentication & Group Sync**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**         | **API / Route**                                      |
| --------------- | ------------ | --------- | --------------------- | ---------------------------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 2   | ab_user, ab_user_role | POST /api/v1/auth/login (LDAP handled transparently) |

**⚑ Depends on:** AUTH-003 (user upsert), AUTH-010 (role sync)

| **⚙️ Backend - Description**
- LDAP bind flow: service bind → search user DN → user bind → fetch group memberships → map groups to ab_role → upsert ab_user → issue JWT. LDAP_FALLBACK_TO_LOCAL for unreachable LDAP.
**🔄 Request Flow**
1. Bind service account → search user → bind as user → fetch groups → map roles → upsert → JWT
**⚙️ Go Implementation**
1. go-ldap/ldap v3
2. l.Bind(serviceDN,pass) → l.Search(userFilter) → l.Bind(userDN,userPass) → l.Search(groupFilter)
3. GORM.FirstOrCreate+role sync | **✅ Acceptance Criteria**
- Valid LDAP creds → JWT + roles synced.
- Invalid → 401.
- Group "superset-admins" → Admin role.
**⚠️ Error Responses**
- 401 - LDAP auth failure.
- 502 - LDAP unreachable. | **🖥️ Frontend Specification**
**📍 Route & Page**
/login (same form, LDAP is backend-transparent)
**🧩 shadcn/ui Components**
- No additional component - same login form as AUTH-003
- Alert (info) - shown if LDAP configured: "Sign in with your corporate credentials"
**📦 State & Data Fetching**
- No frontend state difference - response is same JWT as password login
**✨ UX Behaviors**
- If LDAP_AUTH_ENABLED: show info Alert on login page explaining corporate credentials.
- Username field placeholder changes to "Corporate username or email".
**🌐 API Calls (TanStack Query)**
1. Same POST /api/v1/auth/login - backend detects LDAP vs local internally |
| --- | --- | --- |


**AUTH-015** - **User Profile & Password Management**

| **Dependency**  | **Priority** | **Phase** | **DB Tables**           | **API / Route**                                                                 |
| --------------- | ------------ | --------- | ----------------------- | ------------------------------------------------------------------------------- |
| **⚠ DEPENDENT** | **P1**       | Phase 2   | ab_user, user_attribute | GET/PUT /api/v1/me · PUT /api/v1/me/password · GET/PUT/DELETE /api/v1/users/:id |

**⚑ Depends on:** AUTH-003 (user must exist and be active)

| **⚙️ Backend - Description**
- User profile view/edit. Password change (verify current, complexity check, hash, invalidate refresh tokens). Admin user list/deactivate. Avatar upload to object storage.
**🔄 Request Flow**
1. GET /me → join user+user_attribute → return. PUT password → bcrypt verify → hash new → update → redis delete all refresh tokens
**⚙️ Go Implementation**
1. GORM.Preload("UserAttribute").Preload("Roles").First(&user,uid)
2. bcrypt.CompareHashAndPassword → bcrypt.GenerateFromPassword → GORM.Update → redis.Del refresh pattern | **✅ Acceptance Criteria**
- GET /me → full profile.
- Wrong current_password → 400.
- Password change → all sessions invalidated.
- DELETE /users/:id → active=false.
**⚠️ Error Responses**
- 400 - Wrong current password.
- 403 - Non-admin on other user.
- 422 - Weak new password. | **🖥️ Frontend Specification**
**📍 Route & Page**
/settings/profile · /settings/users · /settings/users/:id
**🧩 shadcn/ui Components**
- - Profile Page (/settings/profile) -
- Tabs [Profile, Security, Preferences] - main page structure
- Card - section containers within each tab
- Avatar + AvatarImage + AvatarFallback - profile picture display
- Button (Upload Avatar) - triggers hidden Input[type=file]
- Form + Input - first_name, last_name, email fields
- Form + Input[type=password] × 3 - current, new, confirm on Security tab
- Button ("Change Password") - separate mutation from profile save
- - User Management (/settings/users) - Admin only -
- DataTable - columns: Name, Email, Roles, Status, Last Login, Actions
- Sheet - user detail slideout (edit roles, view profile)
- AlertDialog - deactivate user confirmation
- Badge (Active/Inactive) - status column
- Select - per-row status toggle (Active/Inactive)
**📦 State & Data Fetching**
- useQuery({ queryKey:["me"] }) - profile data
- useMutation({ mutationFn: api.updateProfile }) - profile form
- useMutation({ mutationFn: api.changePassword, onSuccess: ()=>{ authStore.clearAuth(); navigate("/login?passwordChanged=true") } })
- useQuery({ queryKey:["users"] }) - admin user list
- useMutation({ mutationFn: api.deactivateUser, onSuccess: ()=>queryClient.invalidateQueries(["users"]) })
**✨ UX Behaviors**
- Avatar: click avatar → hidden Input[type=file accept="image/*"] → preview in Avatar before upload → Button "Save" uploads.
- Profile form: auto-populates from /me response. "Save Changes" Button enabled only when isDirty.
- Password change: after success → forced logout with Toast "Password changed. Please sign in again."
- Users table: Status Badge clickable → AlertDialog "Deactivate {name}? They will be signed out immediately."
- User search: Input with Search icon above DataTable → client-side filter on name/email.
- Last Login column: shadcn Tooltip showing exact datetime, cell shows relative ("3 days ago").
**🛡️ Client-Side Validation**
- New password: same Zod schema as registration (min 12, complexity).
- Confirm password must match new password.
- Avatar: accept image/* only, max 2MB client-side before upload.
**♿ Accessibility (a11y)**
- Avatar upload button: aria-label="Upload profile picture".
- Password fields: autocomplete="current-password" and "new-password".
**🌐 API Calls (TanStack Query)**
1. useQuery({ queryKey:["me"], queryFn: ()=>fetch("/api/v1/me").then(r=>r.json()) })
2. useMutation({ mutationFn: (pwd)=>fetch("/api/v1/me/password",{method:"PUT",body:JSON.stringify(pwd)}) })
3. useQuery({ queryKey:["users"], queryFn: ()=>fetch("/api/v1/users").then(r=>r.json()) }) |
| --- | --- | --- |


## **Requirements Summary**

| **ID**   | **Name**                                 | **Priority** | **Dep**       | **FE Route**                                                               | **Endpoint(s)**                                                                                                          | **Phase** |
| -------- | ---------------------------------------- | ------------ | ------------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ | --------- |
| AUTH-001 | User Self-Registration                   | P0           | ✓ INDEPENDENT | /register                                                                  | POST /api/v1/auth/register                                                                                               | Phase 1   |
| AUTH-002 | Email Verification & Account Activation  | P0           | ✓ INDEPENDENT | /auth/verify (handles the email link click)                                | GET /api/v1/auth/verify?hash=                                                                                            | Phase 1   |
| AUTH-003 | Login with Username / Password           | P0           | ✓ INDEPENDENT | /login                                                                     | POST /api/v1/auth/login                                                                                                  | Phase 1   |
| AUTH-004 | JWT Middleware & Token Validation        | P0           | ✓ INDEPENDENT | N/A - frontend-side token management via Zustand + axios/fetch interceptor | Internal middleware - all protected routes                                                                               | Phase 1   |
| AUTH-005 | Refresh Token Rotation                   | P0           | ✓ INDEPENDENT | N/A - called by interceptor, no UI page                                    | POST /api/v1/auth/refresh                                                                                                | Phase 1   |
| AUTH-006 | Logout & Token Revocation                | P0           | ✓ INDEPENDENT | Triggered from /settings or header dropdown - no dedicated page            | POST /api/v1/auth/logout                                                                                                 | Phase 1   |
| AUTH-007 | Role CRUD Management                     | P0           | ✓ INDEPENDENT | /settings/roles                                                            | GET/POST /api/v1/roles · PUT/DELETE /api/v1/roles/:id                                                                    | Phase 1   |
| AUTH-008 | Permission & View Menu Management        | P0           | ✓ INDEPENDENT | /settings/permissions                                                      | GET/POST /api/v1/permissions · GET/POST /api/v1/view-menus · GET/POST/DELETE /api/v1/permission-views                    | Phase 1   |
| AUTH-009 | Assign Permissions to Role (RBAC Matrix) | P0           | ✓ INDEPENDENT | /settings/roles/:id/permissions                                            | PUT /api/v1/roles/:id/permissions · POST /api/v1/roles/:id/permissions/add · DELETE /api/v1/roles/:id/permissions/:pv_id | Phase 1   |
| AUTH-010 | Assign Roles to User                     | P0           | ✓ INDEPENDENT | /settings/users/:id (user detail page, roles section)                      | PUT /api/v1/users/:id/roles · GET /api/v1/users/:id/roles                                                                | Phase 1   |
| AUTH-011 | RBAC Permission Check Middleware         | P0           | ✓ INDEPENDENT | N/A - handled via route guards in React Router                             | Internal middleware - wraps all protected endpoints                                                                      | Phase 1   |
| AUTH-012 | Multi-Tenant Context Injection           | P0           | ⚠ DEPENDENT   | N/A - org_id is transparent to the frontend                                | Internal middleware - all DB queries                                                                                     | Phase 1   |
| AUTH-013 | OAuth2 Federation (Google / Okta)        | P1           | ⚠ DEPENDENT   | /login (OAuth buttons) + /auth/oauth2/callback (redirect target)           | GET /api/v1/auth/oauth2/:provider/authorize · GET /api/v1/auth/oauth2/:provider/callback                                 | Phase 2   |
| AUTH-014 | LDAP Authentication & Group Sync         | P1           | ⚠ DEPENDENT   | /login (same form, LDAP is backend-transparent)                            | POST /api/v1/auth/login (LDAP handled transparently)                                                                     | Phase 2   |
| AUTH-015 | User Profile & Password Management       | P1           | ⚠ DEPENDENT   | /settings/profile · /settings/users · /settings/users/:id                  | GET/PUT /api/v1/me · PUT /api/v1/me/password · GET/PUT/DELETE /api/v1/users/:id                                          | Phase 2   |