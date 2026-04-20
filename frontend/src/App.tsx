import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import RegisterPage from "@/pages/register/RegisterPage";
import RegisterSuccessPage from "@/pages/register/RegisterSuccessPage";
import VerifyPage from "@/pages/auth/VerifyPage";
import LoginPage from "@/pages/auth/LoginPage";
import HomePage from "@/pages/home/HomePage";
import RolesPage from "@/pages/admin/RolesPage";
import PermissionsPage from "@/pages/admin/PermissionsPage";
import RolePermissionsPage from "@/pages/admin/RolePermissionsPage";
import UserRolesPage from "@/pages/admin/UserRolesPage";
import UsersPage from "@/pages/admin/UsersPage";
import AdminLayout from "@/pages/admin/AdminLayout";
import AdminDashboardPage from "@/pages/admin/AdminDashboardPage";
import DatabasesPage from "@/pages/admin/DatabasesPage";
import CreateDatabasePage from "@/pages/admin/CreateDatabasePage";
import EditDatabasePage from "@/pages/admin/EditDatabasePage";
import CreateDatasetPage from "@/pages/datasets/CreateDatasetPage";
import DatasetsPage from "@/pages/admin/DatasetsPage";
import { ProtectedRoute } from "@/components/ProtectedRoute";
import { Toaster } from "@/components/ui/sonner";

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Public routes — LoginPage redirects away if a valid session exists */}
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/register/success" element={<RegisterSuccessPage />} />
        <Route path="/auth/verify" element={<VerifyPage />} />

        {/* Protected routes */}
        <Route element={<ProtectedRoute />}>
          <Route path="/" element={<HomePage />} />
        </Route>

        {/* Admin routes (authorization enforced by backend APIs) */}
        <Route element={<ProtectedRoute />}>
          <Route path="/admin" element={<AdminLayout />}>
            <Route index element={<Navigate to="dashboard" replace />} />
            <Route path="dashboard" element={<AdminDashboardPage />} />
            <Route path="settings/roles" element={<RolesPage />} />
            <Route path="settings/roles/:id/permissions" element={<RolePermissionsPage />} />
            <Route path="settings/users" element={<UsersPage />} />
            <Route path="settings/users/:id" element={<UserRolesPage />} />
            <Route path="settings/databases" element={<DatabasesPage />} />
            <Route path="settings/databases/new" element={<CreateDatabasePage />} />
            <Route path="settings/databases/:id" element={<EditDatabasePage />} />
            <Route path="settings/datasets" element={<DatasetsPage />} />
            <Route path="settings/datasets/new" element={<CreateDatasetPage />} />
            <Route path="settings/datasets/:id/edit" element={<CreateDatasetPage />} />
            <Route path="settings/permissions" element={<PermissionsPage />} />
          </Route>
        </Route>

        {/* Fallback */}
        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>

      {/* Session-expiry toasts */}
      <Toaster />
    </BrowserRouter>
  );
}
