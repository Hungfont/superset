import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import RegisterPage from "./RegisterPage";

// Mock the api module so no real HTTP calls are made
vi.mock("@/api/auth", () => ({
  authApi: {
    register: vi.fn(),
  },
}));

import { authApi } from "@/api/auth";

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { mutations: { retry: 0 } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={["/register"]}>
        <Routes>
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/register/success" element={<div>Success page</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

function getPasswordInput() {
  return screen.getByLabelText("Password");
}
function getConfirmPasswordInput() {
  return screen.getByLabelText("Confirm password");
}

async function fillValidForm(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/first name/i), "John");
  await user.type(screen.getByLabelText(/last name/i), "Doe");
  await user.type(screen.getByLabelText(/username/i), "johndoe");
  await user.type(screen.getByLabelText(/email/i), "john@example.com");
  await user.type(getPasswordInput(), "StrongP@ss1!");
  await user.type(getConfirmPasswordInput(), "StrongP@ss1!");
}

describe("RegisterPage", () => {
  const user = userEvent.setup();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders all form fields", () => {
    renderPage();
    expect(screen.getByLabelText(/first name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/last name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/username/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
    expect(getPasswordInput()).toBeInTheDocument();
    expect(getConfirmPasswordInput()).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /create account/i })).toBeInTheDocument();
  });

  it("shows field-level validation errors on empty submit", async () => {
    renderPage();
    await user.click(screen.getByRole("button", { name: /create account/i }));
    await waitFor(() => {
      expect(screen.getByText(/first name is required/i)).toBeInTheDocument();
    });
  });

  it("shows mismatched password error", async () => {
    renderPage();
    await user.type(getPasswordInput(), "StrongP@ss1!");
    await user.type(getConfirmPasswordInput(), "DifferentPass1!");
    await user.click(screen.getByRole("button", { name: /create account/i }));
    await waitFor(() => {
      expect(screen.getByText(/passwords do not match/i)).toBeInTheDocument();
    });
  });

  it("navigates to /register/success on successful submit", async () => {
    vi.mocked(authApi.register).mockResolvedValue({ message: "Verification email sent" });
    renderPage();
    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: /create account/i }));
    await waitFor(() => {
      expect(screen.getByText(/success page/i)).toBeInTheDocument();
    });
  });

  it("shows server error alert on API failure", async () => {
    vi.mocked(authApi.register).mockRejectedValue(new Error("email already registered"));
    renderPage();
    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: /create account/i }));
    await waitFor(() => {
      expect(screen.getByRole("alert")).toBeInTheDocument();
      expect(screen.getByText(/email already registered/i)).toBeInTheDocument();
    });
  });

  it("disables submit button while mutation is pending", async () => {
    // Never resolves — simulates in-flight request
    vi.mocked(authApi.register).mockReturnValue(new Promise(() => {}));
    renderPage();
    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: /create account/i }));
    await waitFor(() => {
      expect(screen.getByRole("button", { name: /create account/i })).toBeDisabled();
    });
  });
});
