---
paths:
  - "src/**/*.tsx"
  - "src/**/*.jsx"
  - "components/**/*.tsx"
---

# React/TSX Anti-Pattern Enforcement

## Before Writing a Component
- [ ] Does this component already exist? Search `components/` first
- [ ] Is this >1 responsibility? Split it
- [ ] Are props >5? Consider grouping into an object or Context

## Before Writing a Hook
- [ ] Does `useEffect` have all dependencies listed?
- [ ] Is this derived state? Use `useMemo` not `useState + useEffect`
- [ ] Is async handling correct? (no async directly in useEffect)

## Before Submitting
- [ ] No `any` types without comment explaining why
- [ ] No inline objects/arrays/functions as props in loops
- [ ] No index as key in dynamic lists
- [ ] Exported functions have explicit return types

### Anti-Pattern Sample: 

Duplicate Create and Update Pages
## Bad: Split pages with duplicated form and logic

```tsx
// CreateUserPage.tsx
export function CreateUserPage(): JSX.Element {
  const form = useForm<UserFormValues>({ resolver: zodResolver(userSchema) })
  const createMutation = useMutation({ mutationFn: usersApi.create })

  return <UserForm form={form} onSubmit={(values) => createMutation.mutate(values)} />
}

// UpdateUserPage.tsx
export function UpdateUserPage(): JSX.Element {
  const { id } = useParams<{ id: string }>()
  const form = useForm<UserFormValues>({ resolver: zodResolver(userSchema) })
  const updateMutation = useMutation({ mutationFn: (values) => usersApi.update(Number(id), values) })

  return <UserForm form={form} onSubmit={(values) => updateMutation.mutate(values)} />
}
```

## Good: One reusable upsert page and shared form

```tsx
type UserFormMode = 'create' | 'update'

export function UserUpsertPage(): JSX.Element {
  const { id } = useParams<{ id?: string }>()
  const mode: UserFormMode = id ? 'update' : 'create'
  const userId = id ? Number(id) : null

  const form = useForm<UserFormValues>({
    resolver: zodResolver(userSchema),
    defaultValues: DEFAULT_USER_FORM_VALUES,
  })

  const createMutation = useMutation({ mutationFn: usersApi.create })
  const updateMutation = useMutation({
    mutationFn: (values: UserFormValues) => usersApi.update(userId as number, values),
  })

  const handleSubmit = (values: UserFormValues): void => {
    if (mode === 'create') {
      createMutation.mutate(values)
      return
    }
    updateMutation.mutate(values)
  }

  return <UserForm mode={mode} form={form} onSubmit={handleSubmit} />
}
```

Use separate pages only if create and update have materially different fields, permissions, or workflow.