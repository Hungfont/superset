<!-- Generated: 2026-04-13 | Files scanned: ~15 | Token estimate: ~400 -->

# Frontend Codemap

**Entry Point:** `frontend/src/main.tsx` (Vite)  
**Stack:** React + TypeScript + Tailwind + shadcn/ui + Zustand

## Page Tree

```
/            → pages/home/
/login       → pages/auth/     (login form)
/register    → pages/register/ (registration form)
```

## Key Files

| File | Purpose |
|------|---------|
| `src/api/auth.ts` | Raw API calls: register, login, refresh |
| `src/lib/api/client.ts` | Axios/fetch client, base URL, interceptors |
| `src/stores/authStore.ts` | Zustand auth state (user, tokens, actions) |
| `src/hooks/useLogin.ts` | Login mutation — calls API, updates store |
| `src/hooks/useRegister.ts` | Register mutation |
| `src/hooks/useTokenRefresh.ts` | Silent token refresh on expiry |
| `src/hooks/use-toast.ts` | Toast notification hook (shadcn) |
| `src/lib/validations/login.ts` | Zod schema for login form |
| `src/lib/validations/register.ts` | Zod schema for register form |
| `src/lib/utils.ts` | Tailwind `cn()` utility |
| `src/components/ui/` | shadcn/ui component library |
| `src/test/setup.ts` | Vitest test environment setup |

## State Management

```
authStore (Zustand)
  ├── user: UserContext | null
  ├── accessToken: string | null
  ├── login(credentials) → calls API → sets tokens + user
  ├── logout() → clears state
  └── refresh() → useTokenRefresh → rotates access token
```

## Config Files

| File | Purpose |
|------|---------|
| `vite.config.ts` | Build config, dev proxy |
| `tailwind.config.js` | Theme, content paths |
| `components.json` | shadcn registry config |
| `tsconfig.json` | TS strict mode |

## Related

- [architecture.md](architecture.md) — API endpoints consumed
- [dependencies.md](dependencies.md) — npm packages
