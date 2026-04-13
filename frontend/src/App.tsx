import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import RegisterPage from "@/pages/register/RegisterPage";
import RegisterSuccessPage from "@/pages/register/RegisterSuccessPage";
import VerifyPage from "@/pages/auth/VerifyPage";
import LoginPage from "@/pages/auth/LoginPage";
import HomePage from "@/pages/home/HomePage";
import RolesPage from "@/pages/settings/RolesPage";
import RolePermissionsPage from "@/pages/settings/RolePermissionsPage";
import AdminLayout from "@/pages/settings/AdminLayout";
import AdminDashboardPage from "@/pages/settings/AdminDashboardPage";
import { ProtectedRoute } from "@/components/ProtectedRoute";
import { Toaster } from "@/components/ui/toaster";

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

        {/* Admin-only routes */}
        <Route element={<ProtectedRoute requiredRole="Admin" />}>
          <Route path="/dashboard" element={<AdminLayout />}>
            <Route index element={<AdminDashboardPage />} />
            <Route path="/settings/roles" element={<RolesPage />} />
            <Route path="/settings/roles/:id/permissions" element={<RolePermissionsPage />} />
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
