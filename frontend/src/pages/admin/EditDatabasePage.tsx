import CreateDatabasePage from "./CreateDatabasePage";

/**
 * Edit Database Page
 * 
 * Reuses CreateDatabasePage component with mode detection via URL params.
 * When accessed at /admin/settings/databases/:id, CreateDatabasePage
 * automatically switches to edit mode and fetches existing database data.
 */
export default function EditDatabasePage() {
  return <CreateDatabasePage />;
}
