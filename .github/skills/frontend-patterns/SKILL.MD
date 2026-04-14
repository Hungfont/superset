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

  return (
    <>
      <Button onClick={() => { form.reset(); setIsOpen(true) }}>Open</Button>

      <Dialog
        open={isOpen}
        onOpenChange={(open) => { setIsOpen(open); if (!open) form.reset() }}
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

## Forms — React Hook Form + Zod + shadcn Form

**MANDATORY pattern** for all forms. Never use manual `useState` for form fields.

```tsx
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  Form, FormField, FormItem, FormLabel, FormControl, FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'


export function ExampleForm({ onSubmit }: { onSubmit: (v: FormValues) => void }) {
// 1. registerSchema defines validation rules and error messages in lib/validation.ts
  const form = useForm<RegisterFormValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: {
      first_name: "",
      last_name: "",
      username: "",
      email: "",
      password: "",
      confirmPassword: "",
    },
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

// Dialog onOpenChange — always reset on close
onOpenChange={(open) => { setIsOpen(open); if (!open) form.reset() }}
```

**Rules:**
- Schema must be a `z.object({...})` — not `z.string()` directly
- `type FormValues = z.infer<typeof schema>` — infer types from schema, never duplicate
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
      <Button variant="outline" onClick={() => void rolesQuery.refetch()}>Retry</Button>
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
