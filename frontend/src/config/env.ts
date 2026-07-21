/**
 * Import `env` anywhere in app code instead of using `import.meta.env`
 * directly. Validation of environment variables already ran, so every field
 * here is guaranteed present and well-formed.
 */
import { parseEnv } from "./env.schema";

export const env = parseEnv(import.meta.env);
export type { Env } from "./env.schema";
