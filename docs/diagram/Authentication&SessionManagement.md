sequenceDiagram
    autonumber
    actor User
    participant Browser
    participant Flask as Flask API (FAB)
    participant SecMgr as Security Manager
    participant IdP as OAuth / LDAP Provider
    participant MetaDB as Metadata DB (ab_user)
    participant Redis as Redis Session Store

    rect rgb(219,234,254)
        Note over User,Redis: Flow A — Database (username/password) Login
        User->>Browser: Submit login form
        Browser->>Flask: POST /api/v1/security/login {username, password, provider:"db"}
        Flask->>SecMgr: authenticate(username, password)
        SecMgr->>MetaDB: SELECT * FROM ab_user WHERE username=?
        MetaDB-->>SecMgr: User record
        SecMgr->>SecMgr: bcrypt.verify(password, hash)
        SecMgr-->>Flask: User object
        Flask->>MetaDB: UPDATE ab_user SET last_login, login_count
        Flask->>Redis: SET session token (JWT)
        Flask-->>Browser: {access_token, refresh_token}
        Browser->>Browser: Store JWT in memory
    end

    rect rgb(220,252,231)
        Note over User,Redis: Flow B — OAuth2 / OIDC Login
        User->>Browser: Click "Login with OAuth"
        Browser->>Flask: GET /oauth-authorized/{provider}
        Flask->>IdP: Redirect → Authorization endpoint
        IdP-->>User: Login page
        User->>IdP: Credentials
        IdP-->>Flask: Authorization code
        Flask->>IdP: POST /token (code exchange)
        IdP-->>Flask: {access_token, id_token, profile}
        Flask->>SecMgr: oauth_user_info(provider, token)
        SecMgr->>MetaDB: Upsert ab_user (email, name)
        SecMgr->>MetaDB: Assign default roles → ab_user_role
        MetaDB-->>SecMgr: User record
        Flask->>Redis: SET session
        Flask-->>Browser: Redirect + Set-Cookie session
    end

    rect rgb(252,231,243)
        Note over User,Redis: Flow C — Token Refresh
        Browser->>Flask: POST /api/v1/security/refresh (refresh_token)
        Flask->>SecMgr: validate_refresh_token()
        SecMgr->>MetaDB: Verify user still active
        MetaDB-->>SecMgr: OK
        Flask-->>Browser: New {access_token}
    end

    rect rgb(255,237,213)
        Note over User,Redis: Flow D — Logout
        User->>Browser: Click Logout
        Browser->>Flask: DELETE /api/v1/security/logout
        Flask->>Redis: DEL session key
        Flask-->>Browser: 200 OK, clear cookie
    end