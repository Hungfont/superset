import { DATABASE_TYPE_OPTIONS } from "@/lib/validations/database";
import type { CreateDatabaseFormValues, UpdateDatabaseFormValues } from "@/lib/validations/database";

/**
 * Build SQLAlchemy URI from form values
 */
export function buildSQLAlchemyURI(
  values: CreateDatabaseFormValues | UpdateDatabaseFormValues
): string {
  const selectedType = DATABASE_TYPE_OPTIONS.find((opt) => opt.value === values.db_type);
  const driver = selectedType?.driver ?? values.db_type;
  const username = encodeURIComponent(values.username);
  
  // Handle password - could be "***" for update mode (keep existing)
  let password: string;
  if ("password" in values && !values.password) {
    // Empty password in update mode means keep existing
    password = "***";
  } else {
    password = encodeURIComponent(values.password || "");
  }

  return `${driver}://${username}:${password}@${values.host}:${values.port}/${values.database}`;
}

/**
 * Mask password in SQLAlchemy URI for display
 */
export function maskSQLAlchemyURI(sqlalchemyURI: string): string {
  return sqlalchemyURI.replace(/:\/\/([^:]+):([^@]+)@/, "://$1:***@");
}

/**
 * Parse SQLAlchemy URI back to form values
 */
export function parseDatabaseURI(sqlalchemyURI: string) {
  try {
    const parsed = new URL(sqlalchemyURI);
    const dbType = parsed.protocol.replace(":", "").toLowerCase();

    return {
      dbType,
      host: parsed.hostname || "",
      port: parsed.port ? Number(parsed.port) : 5432,
      database: parsed.pathname.replace(/^\//, ""),
      username: decodeURIComponent(parsed.username || ""),
    };
  } catch (err) {
    console.error("Failed to parse database URI:", err);
    return {
      dbType: "",
      host: "",
      port: 5432,
      database: "",
      username: "",
    };
  }
}

/**
 * Get default port for database type
 */
export function getDefaultPort(dbType: string): number {
  const option = DATABASE_TYPE_OPTIONS.find((opt) => opt.value === dbType);
  return option?.defaultPort ?? 5432;
}

/**
 * Get driver for database type
 */
export function getDriver(dbType: string): string {
  const option = DATABASE_TYPE_OPTIONS.find((opt) => opt.value === dbType);
  return option?.driver ?? dbType;
}
