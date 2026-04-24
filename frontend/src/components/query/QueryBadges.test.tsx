import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { CacheBadge, RLSBadge, QueryStatusBadge, RunButton } from "./QueryBadges";

describe("CacheBadge", () => {
  it("shows cached badge when from_cache is true", () => {
    const onForceRefresh = vi.fn();
    render(
      <CacheBadge
        fromCache={true}
        durationMs={3}
        onForceRefresh={onForceRefresh}
      />
    );

    expect(screen.getByText(/Cached \(3ms\)/)).toBeInTheDocument();
  });

  it("shows live badge when from_cache is false", () => {
    render(
      <CacheBadge
        fromCache={false}
        durationMs={234}
        onForceRefresh={vi.fn()}
      />
    );

    expect(screen.getByText(/Live \(234ms\)/)).toBeInTheDocument();
  });

  it("calls onForceRefresh when clicked", () => {
    const onForceRefresh = vi.fn();
    render(
      <CacheBadge
        fromCache={true}
        durationMs={3}
        onForceRefresh={onForceRefresh}
      />
    );

    fireEvent.click(screen.getByRole("button"));
    expect(onForceRefresh).toHaveBeenCalledTimes(1);
  });

  it("returns null when no duration provided", () => {
    const { container } = render(
      <CacheBadge
        fromCache={false}
        onForceRefresh={vi.fn()}
      />
    );

    expect(container.firstChild).toBeNull();
  });
});

describe("RLSBadge", () => {
  it("shows badge when RLS is applied", () => {
    render(
      <RLSBadge
        executedSql="SELECT * FROM orders WHERE org_id = 42"
        originalSql="SELECT * FROM orders"
      />
    );

    expect(screen.getByText(/RLS Active/)).toBeInTheDocument();
  });

  it("returns null when RLS is not applied", () => {
    const { container } = render(
      <RLSBadge
        executedSql="SELECT * FROM orders"
        originalSql="SELECT * FROM orders"
      />
    );

    expect(container.firstChild).toBeNull();
  });
});

describe("QueryStatusBadge", () => {
  it("shows running badge when status is running", () => {
    render(<QueryStatusBadge status="running" />);
    expect(screen.getByText(/Running\.\.\./)).toBeInTheDocument();
  });

  it("shows done badge when status is success", () => {
    render(<QueryStatusBadge status="success" />);
    expect(screen.getByText(/Done/)).toBeInTheDocument();
  });

  it("shows failed badge when status is error", () => {
    render(<QueryStatusBadge status="error" />);
    expect(screen.getByText(/Failed/)).toBeInTheDocument();
  });

  it("returns null when status is idle", () => {
    const { container } = render(<QueryStatusBadge status="idle" />);
    expect(container.firstChild).toBeNull();
  });
});

describe("RunButton", () => {
  it("shows Run button when not running", () => {
    render(
      <RunButton
        onClick={vi.fn()}
        disabled={false}
        isRunning={false}
      />
    );

    expect(screen.getByText(/Run/)).toBeInTheDocument();
  });

  it("shows loading state when running", () => {
    render(
      <RunButton
        onClick={vi.fn()}
        disabled={false}
        isRunning={true}
      />
    );

    expect(screen.getByText(/Running\.\.\./)).toBeInTheDocument();
  });
});