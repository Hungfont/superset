---
name: frontend-patterns
description: Frontend development patterns for React 18, TanStack Query v5, React Hook Form + Zod, Zustand, shadcn/ui, TanStack Table v8, and React Router v6.
origin: ECC
---

# Frontend Development Patterns

Patterns for this project's stack: React 18 + TypeScript · Vite · TanStack Query v5 · React Hook Form + Zod · Zustand · shadcn/ui (Radix UI) · TanStack Table v8 · React Router v6 · Tailwind CSS · Lucide React.

## When to Activate

- Building React components (composition, props, controlled dialogs)
- Managing server state (TanStack Query useQuery / useMutation)
- Managing client/UI state (Zustand, useState, useReducer)
- Working with forms (React Hook Form + Zod + shadcn Form components)
- Building data tables (TanStack Table v8 + shadcn DataTable pattern)
- Handling routing (React Router v6 nested routes, ProtectedRoute)
- Optimizing performance (memoization, virtualization, code splitting)
- Building accessible UI with shadcn/ui (Dialog, Sheet, AlertDialog)

---

## Component Patterns

### Composition Over Inheritance

Always use `shadcn/ui` primitives. Never build custom styled divs/buttons/inputs.

### Shared UI Primitives (`src/components/ui`)

Treat `src/components/ui` as the project’s shared primitive layer.

- Reuse existing primitives from `src/components/ui` before creating new wrappers.
- If a primitive is available from shadcn/ui, generate or replace it through the shadcn workflow instead of hand-rolling a local variant.
- Keep exported component names and composition patterns aligned with shadcn conventions so pages can swap primitives without changing calling code.
- Prefer composing pages and features from these primitives rather than styling bespoke elements directly in page components.
- When a primitive needs browser-specific behavior, centralize the compatibility fix in the shared component or test setup instead of patching each consumer.

```tsx
// CORRECT: shadcn composition
import { Card, CardHeader, CardContent, CardFooter } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

interface RoleCardProps {
  name: string
  userCount: number
  onEdit: () => void
}

export function RoleCard({ name, userCount, onEdit }: RoleCardProps) {
  return (
    <Card>
      <CardHeader>{name}</CardHeader>
      <CardContent>
        <Badge variant="secondary">{userCount} users</Badge>
      </CardContent>
      <CardFooter>
        <Button variant="outline" onClick={onEdit}>Edit</Button>
      </CardFooter>
    </Card>
  )
}
```

### Controlled Dialog Pattern (shadcn/ui)

Use controlled `open` + `onOpenChange` — never `DialogTrigger` for programmatic dialogs.
Always reset form state on close via `onOpenChange`.

```tsx
import { useState } from 'react'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog'

export function ExamplePage() {
  const [isOpen, setIsOpen] = useState(false)
  const form = useForm(...)

  // Open for create — reset to empty defaults
  const handleOpenCreate = () => {
    form.reset({ name: '' })
    setIsOpen(true)
  }

  const onOpenChange = (open: boolean) => {
    setIsOpen(open)
    if (!open) {
      form.reset()  // also reset on close to clear any unsaved changes
    }
  }

  return (
    <>
      <Button onClick={handleOpenCreate}>Open</Button>

      <Dialog
        open={isOpen}
        onOpenChange={onOpenChange}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>...</DialogTitle>
            <DialogDescription>...</DialogDescription>
          </DialogHeader>
          {/* form body */}
        </DialogContent>
      </Dialog>
    </>
  )
}
```

**Pitfall — missing animation classes on DialogOverlay block clicks after close.**
`DialogOverlay` must have `data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0`.
Without them, the exit keyframes animate to `opacity: 1` (default), leaving the overlay
opaque for the full animation duration and blocking pointer events on underlying buttons.

---

## Data Fetching — TanStack Query v5

### useQuery — Read

```tsx
import { useQuery } from '@tanstack/react-query'
import { rolesApi, type Role } from '@/api/roles'

export function useRoles() {
  return useQuery({
    queryKey: ['roles'],
    queryFn: rolesApi.getRoles,
  })
}

// In component
const rolesQuery = useRoles()
const roles = rolesQuery.data ?? []
```

### useMutation — Write with Optimistic Update

```tsx
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useToast } from '@/hooks/use-toast'

const queryClient = useQueryClient()
const { success, error } = useToast()

const createMutation = useMutation({
  mutationFn: rolesApi.createRole,
  onMutate: async (payload) => {
    await queryClient.cancelQueries({ queryKey: ['roles'] })
    const previous = queryClient.getQueryData<Role[]>(['roles']) ?? []
    const optimistic: Role = { id: -Date.now(), ...payload, user_count: 0, permission_count: 0, built_in: false }
    queryClient.setQueryData<Role[]>(['roles'], [...previous, optimistic])
    return { previous }
  },
  onError: (_err, _payload, context) => {
    if (context?.previous) queryClient.setQueryData(['roles'], context.previous)
    error('Failed')
  },
  onSuccess: () => {
    success('Created')
  },
  onSettled: () => {
    queryClient.invalidateQueries({ queryKey: ['roles'] })
  },
})
```

**Rules:**
- `useQuery` for GET — never fetch inside `useEffect`
- `useMutation` for POST / PUT / DELETE — never call API directly in handlers
- `queryKey` arrays are the cache key — be consistent
- `onMutate` + rollback in `onError` for optimistic updates
- `onSettled` always invalidates to sync with server

---

## API Client Patterns — `src/api`

Keep API access in `src/api` modules. Use a shared request wrapper, typed response envelopes, and a namespaced API object for related endpoints.

**Common pattern:**
- Define a small `ApiEnvelope<T>` type when the backend wraps payloads in `{ data }`.
- Read the access token from `useAuthStore.getState()` inside a helper such as `getAuthHeaders()`.
- Add `Content-Type: application/json` only for write requests that send a JSON body.
- Export typed payload and entity interfaces from the API module so callers do not re-declare them.
- Group related methods under a single object such as `permissionsApi`, `rolesApi`, or `authApi`,etc.
- Return `body.data` from the request wrapper so page components and hooks work with plain typed data.

```ts
interface ApiEnvelope<T> {
  data: T
}

function getAuthHeaders(contentType = false): HeadersInit {
  const accessToken = useAuthStore.getState().accessToken
  return {
    ...(contentType ? { 'Content-Type': 'application/json' } : {}),
    ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
  }
}

export const permissionsApi = {
  async getPermissions(): Promise<Permission[]> {
    const body = await request<ApiEnvelope<Permission[]>>('/api/v1/admin/permissions', {
      method: 'GET',
      credentials: 'include',
      headers: getAuthHeaders(),
    })
    return body.data
  },
}
```

**Rules:**
- Keep all API calls inside `src/api` modules, not in pages or components
- Use a shared request helper for response handling and error normalization
- Prefer one API object per domain so tests and imports stay predictable
- Keep payload and response interfaces near the API methods that use them

---

## Forms — React Hook Form + Zod + shadcn Form

**MANDATORY pattern** for all forms. Never use manual `useState` for form fields.

Define each `zodSchema` and other schema validations in `src/lib/validation/` and export the matching `type` via `z.infer<typeof schema>` from the same file. Page and feature components must import the schema and inferred type; they must not declare Zod schemas or duplicate form types inline.

```tsx
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import {
  Form, FormField, FormItem, FormLabel, FormControl, FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { exampleSchema, type ExampleFormValues, DEFAULT_FORM_VALUES } from '@/lib/validation/example'


export function ExampleForm({ onSubmit }: { onSubmit: (v: ExampleFormValues) => void }) {
  const form = useForm<ExampleFormValues>({
    resolver: zodResolver(exampleSchema),
    defaultValues: DEFAULT_FORM_VALUES,
  });


  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-4">
        <FormField
          control={form.control}
          name="name"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Name</FormLabel>
              <FormControl>
                <Input placeholder="Enter name" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name="email"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Email</FormLabel>
              <FormControl>
                <Input type="email" placeholder="you@example.com" {...field} />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button type="submit">Submit</Button>
      </form>
    </Form>
  )
}
```

### Resetting Form on Dialog Open/Close

```tsx
// Open for create — reset to empty defaults
const handleOpenCreate = () => {
  form.reset({ name: '' })
  setIsOpen(true)
}

// Open for edit — populate with existing values
const handleOpenEdit = (item: Item) => {
  form.reset({ name: item.name })
  setSelected(item)
  setIsOpen(true)
}

const onOpenChange = (open: boolean) => {
    setIsOpen(open)
    if (!open) {
      form.reset()  // also reset on close to clear any unsaved changes
    }
  }


// Dialog onOpenChange — always reset on close
onOpenChange={onOpenChange}
```

**Rules:**
- Schema and `type FormValues = z.infer<typeof schema>` must live in `src/lib/validation/` next to each other
- Page components must import `schema` and inferred types; they must not define Zod objects inline
- `FormControl` wires `id`, `aria-describedby`, `aria-invalid` automatically
- `FormMessage` reads errors from RHF context — no manual error rendering
- Submit handler receives validated `FormValues` — no `event.preventDefault()` needed

---

## Client State — Zustand

Use Zustand for global UI state that survives navigation (auth, sidebar, user prefs).
Use `useState` for local component state (dialog open/close, selected row).

```tsx
import { create } from 'zustand'

interface AuthState {
  user: User | null
  accessToken: string | null
  isAuthenticated: boolean
  setAuth: (user: User, token: string) => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  accessToken: null,
  isAuthenticated: false,
  setAuth: (user, accessToken) => set({ user, accessToken, isAuthenticated: true }),
  clearAuth: () => set({ user: null, accessToken: null, isAuthenticated: false }),
}))

// In components
const { user, isAuthenticated } = useAuthStore()
const setAuth = useAuthStore((s) => s.setAuth)  // subscribe to slice, not whole store
```

**Rules:**
- Store access tokens **in memory** (Zustand), never `localStorage`
- Refresh tokens in HttpOnly cookies only
- Subscribe to slices (`useAuthStore(s => s.user)`) to avoid unnecessary re-renders

---

## Data Tables — TanStack Table v8

```tsx
import { useMemo } from 'react'
import {
  type ColumnDef, flexRender, getCoreRowModel, useReactTable,
} from '@tanstack/react-table'

interface Role { id: number; name: string; user_count: number }

export function RolesTable({ roles }: { roles: Role[] }) {
  const columns = useMemo<ColumnDef<Role>[]>(() => [
    { accessorKey: 'name', header: 'Name' },
    {
      accessorKey: 'user_count',
      header: 'Users',
      cell: ({ row }) => <Badge variant="secondary">{row.original.user_count}</Badge>,
    },
    {
      id: 'actions',
      header: 'Actions',
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" aria-label={`Actions for ${row.original.name}`}>
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem>Edit</DropdownMenuItem>
            <DropdownMenuItem>Delete</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
    },
  ], [])

  const table = useReactTable({ data: roles, columns, getCoreRowModel: getCoreRowModel() })

  return (
    <div className="overflow-hidden rounded-lg border">
      <table className="w-full text-sm">
        <thead className="bg-muted/40">
          {table.getHeaderGroups().map(hg => (
            <tr key={hg.id}>
              {hg.headers.map(h => (
                <th key={h.id} className="px-4 py-3 text-left font-medium">
                  {h.isPlaceholder ? null : flexRender(h.column.columnDef.header, h.getContext())}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody>
          {table.getRowModel().rows.map(row => (
            <tr key={row.id} className="border-t hover:bg-muted/20">
              {row.getVisibleCells().map(cell => (
                <td key={cell.id} className="px-4 py-3">
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
```

**Rules:**
- `columns` wrapped in `useMemo` — column defs are stable references
- Actions column uses `id: 'actions'` (no `accessorKey`) with `aria-label` on trigger
- `Skeleton` for loading, `Alert` for errors — never block full page

---

## Routing — React Router v6

### Protected Route

```tsx
import { Navigate, Outlet } from 'react-router-dom'
import { useAuthStore } from '@/store/auth'

export function ProtectedRoute() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return isAuthenticated ? <Outlet /> : <Navigate to="/login" replace />
}
```

### Nested Layout

```tsx
// App.tsx
<Route element={<ProtectedRoute />}>
  <Route path="/admin" element={<AdminLayout />}>
    <Route index element={<Navigate to="dashboard" replace />} />
    <Route path="dashboard" element={<AdminDashboardPage />} />
    <Route path="settings/roles" element={<RolesPage />} />
  </Route>
</Route>
```

---

## Custom Hooks

### useToast (shadcn Sonner wrapper)

```tsx
import { useToast } from '@/hooks/use-toast'

export function ExampleActions() {
  const { success, error } = useToast()

  const handleSave = async () => {
    try {
      await saveRole()
      success('Role saved')
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error'
      error('Save failed', { description: message })
    }
  }

  return <Button onClick={handleSave}>Save</Button>
}
```

**Rules:**
- Use `useToast()` in components and custom hooks instead of importing `toast` from `sonner` directly.
- Use semantic methods: `success`, `error`, `info`, `warning`.
- Keep page-level failures in `Alert`; use toasts for mutation feedback.

### useToggle

```tsx
export function useToggle(initial = false): [boolean, () => void] {
  const [value, setValue] = useState(initial)
  const toggle = useCallback(() => setValue(v => !v), [])
  return [value, toggle]
}
```

### useDebounce

```tsx
export function useDebounce<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState<T>(value)

  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(id)
  }, [value, delay])

  return debounced
}

// Usage: client-side table filter without API calls on every keystroke
const [search, setSearch] = useState('')
const debouncedSearch = useDebounce(search, 300)
const filtered = useMemo(() => rows.filter(r => r.name.includes(debouncedSearch)), [rows, debouncedSearch])
```

---

## Performance Optimization

### Memoization Rules

```tsx
// useMemo — expensive derivations, stable column defs, filtered/sorted lists
const columns = useMemo<ColumnDef<Role>[]>(() => [...], [navigate])
const sorted = useMemo(() => [...roles].sort((a, b) => a.name.localeCompare(b.name)), [roles])

// useCallback — handlers passed as props to child components
const handleDelete = useCallback((id: number) => deleteMutation.mutate(id), [deleteMutation])

// React.memo — pure display components that receive stable props
export const RoleRow = React.memo(function RoleRow({ role }: { role: Role }) {
  return <tr>...</tr>
})
```

**Do not over-memoize.** `useMemo`/`useCallback` have overhead. Only apply when:
- A computation is measurably expensive, OR
- Referential stability is needed (column defs, callback props to memoized children)

### Code Splitting

```tsx
import { lazy, Suspense } from 'react'
import { Skeleton } from '@/components/ui/skeleton'

const PermissionsMatrix = lazy(() => import('@/pages/settings/PermissionsMatrix'))

<Suspense fallback={<Skeleton className="h-64 w-full" />}>
  <PermissionsMatrix roleId={id} />
</Suspense>
```

### Virtualization for Long Lists

```tsx
import { useVirtualizer } from '@tanstack/react-virtual'

export function VirtualList<T>({ items, renderItem }: { items: T[]; renderItem: (item: T) => React.ReactNode }) {
  const parentRef = useRef<HTMLDivElement>(null)
  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 56,
    overscan: 5,
  })

  return (
    <div ref={parentRef} className="h-[600px] overflow-auto">
      <div style={{ height: virtualizer.getTotalSize(), position: 'relative' }}>
        {virtualizer.getVirtualItems().map(vRow => (
          <div
            key={vRow.index}
            style={{ position: 'absolute', top: 0, transform: `translateY(${vRow.start}px)`, width: '100%' }}
          >
            {renderItem(items[vRow.index])}
          </div>
        ))}
      </div>
    </div>
  )
}
```

---

## Error Handling Patterns

### Page-Level Error State

```tsx
// useQuery error state — show Alert with Retry, never crash
{rolesQuery.isError ? (
  <Alert variant="destructive" role="alert" aria-live="assertive">
    <AlertTitle>Unable to load roles</AlertTitle>
    <AlertDescription className="flex items-center justify-between">
      <span>{rolesQuery.error.message}</span>
      <Button variant="outline">Retry</Button>
    </AlertDescription>
  </Alert>
) : (
  <RolesTable />
)}
```

### Mutation Error via Toast

```tsx
const { success, error } = useToast()

useMutation({
  mutationFn: rolesApi.deleteRole,
  onError: (mutationError) => {
    error('Delete failed', { description: mutationError.message })
  },
  onSuccess: () => {
    success('Role deleted')
  },
})
```

**Rules:**
- `role="alert" aria-live="assertive"` on page-level error `Alert`
- Never use `alert()` or `console.error` for user-visible errors
- Toast (destructive) for mutation errors — inline `Alert` for query errors
- Always provide a Retry button for failed queries

---

## Compound Components

Use for complex UI where child components share implicit state (Tabs, Accordion, etc.).
Prefer shadcn/ui's built-in compound components over custom implementations.

```tsx
// Use shadcn Tabs — not a custom Tab implementation
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'

<Tabs defaultValue="permissions">
  <TabsList>
    <TabsTrigger value="permissions">Permissions</TabsTrigger>
    <TabsTrigger value="users">Users</TabsTrigger>
  </TabsList>
  <TabsContent value="permissions"><PermissionsTab /></TabsContent>
  <TabsContent value="users"><UsersTab /></TabsContent>
</Tabs>
```

Only build custom compound components when no shadcn equivalent exists and the
pattern is used in 3+ places.
