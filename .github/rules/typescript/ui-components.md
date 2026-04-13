---
paths:
  - "**/*.tsx"
  - "**/*.jsx"
---
# UI Components — shadcn/ui MANDATORY

> **CRITICAL RULE**: ALL UI components MUST use shadcn/ui. Custom UI components are FORBIDDEN.

## Mandatory: Use shadcn/ui

This project uses **shadcn/ui** (https://ui.shadcn.com/) as the **only** component library.

- ALWAYS check `npx shadcn@latest search` before writing any UI element
- ALWAYS run `npx shadcn@latest add <component>` to install missing components
- NEVER build custom styled divs, buttons, inputs, modals, or any visual element from scratch

## FORBIDDEN — Custom Components

The following patterns are **BLOCKED** and must be replaced with shadcn equivalents:

```tsx
// WRONG: Custom button
<div className="bg-blue-500 px-4 py-2 rounded cursor-pointer" onClick={...}>Click</div>

// WRONG: Custom input
<input className="border rounded px-3 py-2" />

// WRONG: Custom modal/dialog
<div className="fixed inset-0 z-50 flex items-center justify-center">...</div>

// WRONG: Custom badge
<span className="bg-green-100 text-green-800 px-2 py-1 rounded-full text-xs">Active</span>

// WRONG: Custom loading spinner
<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900" />

// WRONG: Custom empty state
<div className="text-center py-12 text-gray-500">No results found</div>

// WRONG: Custom toast
<div className="fixed bottom-4 right-4 bg-black text-white p-4 rounded">...</div>
```

## CORRECT — shadcn/ui Equivalents

```tsx
// CORRECT: Button
import { Button } from "@/components/ui/button"
<Button variant="default">Click</Button>

// CORRECT: Input
import { Input } from "@/components/ui/input"
<Input placeholder="Enter value" />

// CORRECT: Dialog (modal)
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog"
<Dialog>
  <DialogContent>
    <DialogTitle>Title</DialogTitle>
    ...
  </DialogContent>
</Dialog>

// CORRECT: Badge
import { Badge } from "@/components/ui/badge"
<Badge variant="secondary">Active</Badge>

// CORRECT: Loading skeleton
import { Skeleton } from "@/components/ui/skeleton"
<Skeleton className="h-8 w-8 rounded-full" />

// CORRECT: Empty state
import { Empty } from "@/components/ui/empty"
<Empty>No results found</Empty>

// CORRECT: Toast
import { toast } from "sonner"
toast("Message sent!")
```

## Component Lookup Table

| UI Need | shadcn Component |
|---------|-----------------|
| Button / action | `Button` with variant |
| Text input | `Input` |
| Textarea | `Textarea` |
| Select / dropdown | `Select`, `Combobox` |
| Checkbox | `Checkbox` |
| Radio group | `RadioGroup` |
| Switch / toggle | `Switch` |
| Modal / dialog | `Dialog` |
| Side panel | `Sheet` |
| Bottom sheet | `Drawer` |
| Confirmation | `AlertDialog` |
| Toast / notification | `sonner` → `toast()` |
| Alert / callout | `Alert` |
| Loading placeholder | `Skeleton` |
| Loading spinner | `Spinner` |
| Badge / tag | `Badge` |
| Avatar | `Avatar` + `AvatarFallback` |
| Table | `Table` |
| Card | `Card` + sub-components |
| Tabs | `Tabs`, `TabsList`, `TabsTrigger`, `TabsContent` |
| Accordion | `Accordion` |
| Navigation | `NavigationMenu`, `Breadcrumb` |
| Sidebar | `Sidebar` |
| Pagination | `Pagination` |
| Progress bar | `Progress` |
| Separator / divider | `Separator` |
| Tooltip | `Tooltip` |
| Popover | `Popover` |
| Hover card | `HoverCard` |
| Dropdown menu | `DropdownMenu` |
| Context menu | `ContextMenu` |
| Command palette | `Command` inside `Dialog` |
| Chart | `Chart` (wraps Recharts) |
| Scroll area | `ScrollArea` |
| Resizable panels | `Resizable` |
| OTP input | `InputOTP` |
| Slider | `Slider` |
| Calendar | `Calendar` |
| Date picker | `Calendar` + `Popover` |
| Empty state | `Empty` |

## Form Layout Rules

Forms MUST use `FieldGroup` + `Field`, NEVER raw `div` with spacing:

```tsx
// CORRECT
import { FieldGroup, Field, FieldLabel, FieldDescription } from "@/components/ui/field"
<FieldGroup>
  <Field>
    <FieldLabel htmlFor="email">Email</FieldLabel>
    <Input id="email" />
    <FieldDescription>We'll never share your email.</FieldDescription>
  </Field>
</FieldGroup>

// WRONG
<div className="space-y-4">
  <div>
    <label className="text-sm font-medium">Email</label>
    <input className="border rounded w-full" />
  </div>
</div>
```

## Styling Rules

- **NEVER** use raw Tailwind color classes for component styling (`bg-blue-500`, `text-gray-700`)
- **ALWAYS** use semantic tokens: `bg-primary`, `text-muted-foreground`, `bg-background`
- **NEVER** use `space-x-*` or `space-y-*` — use `flex gap-*` instead
- **ALWAYS** use `cn()` for conditional class names
- **NEVER** manually set `z-index` on overlays (Dialog, Sheet, Popover handle their own stacking)

## Workflow

Before writing any UI code:
1. Run `npx shadcn@latest search -q "<component name>"` to find the component
2. Check if already installed: look in `resolvedPaths.ui` directory
3. Install if missing: `npx shadcn@latest add <component>`
4. Get docs: `npx shadcn@latest docs <component>` and fetch the URLs
5. Implement using the component — no custom markup

## Allowed Exceptions

Custom code is ONLY allowed for:
- **Business logic** (data fetching, state management, calculations)
- **Layout composition** using shadcn layout components
- **Extending** an installed shadcn component with project-specific props (must wrap, not rewrite)
- **Animations** on top of existing shadcn components via `cn()` and Tailwind utilities

If a needed UI pattern does NOT exist in shadcn:
1. First search community registries: `npx shadcn@latest search @magicui -q "<pattern>"`
2. Check other registries: `@tailark`, `@bundui`
3. Only build custom if no registry match exists — and document the reason
