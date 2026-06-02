import { create } from "zustand";
import { type RLSFilter } from "@/api/rlsFilters";

interface RLSStore {
  dialogOpen: boolean;
  editingFilter: RLSFilter | null;
  deleteFilterId: number | null;
  setDialogOpen: (open: boolean) => void;
  setEditingFilter: (filter: RLSFilter | null) => void;
  setDeleteFilterId: (id: number | null) => void;
  openCreate: () => void;
  openEdit: (filter: RLSFilter) => void;
  reset: () => void;
}

export const useRLSStore = create<RLSStore>((set) => ({
  dialogOpen: false,
  editingFilter: null,
  deleteFilterId: null,
  setDialogOpen: (open) => set({ dialogOpen: open, editingFilter: open ? undefined : null }),
  setEditingFilter: (filter) => set({ editingFilter: filter }),
  setDeleteFilterId: (id) => set({ deleteFilterId: id }),
  openCreate: () => set({ dialogOpen: true, editingFilter: null }),
  openEdit: (filter) => set({ dialogOpen: true, editingFilter: filter }),
  reset: () => set({ dialogOpen: false, editingFilter: null, deleteFilterId: null }),
}));
