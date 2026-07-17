import { z } from "zod";

// A string that parses as a URL and uses one of the given protocols.
const urlWithProtocol = (protocols: string[]) =>
    z
        .string({ error: "is required but was not set" })
        .min(1, "must not be empty")
        .superRefine((value, ctx) => {
            let parsed: URL;
            try {
                parsed = new URL(value);
            } catch {
                ctx.addIssue({
                    code: "custom",
                    message: `must be a valid URL (got "${value}")`,
                });
                return;
            }

            const scheme = parsed.protocol.replace(":", "");
            if (!protocols.includes(scheme)) {
                ctx.addIssue({
                    code: "custom",
                    message: `must use protocol ${protocols.map((p) => `${p}://`).join(" or ")} (got "${scheme}://")`,
                });
            }
        });

export const envSchema = z.object({
    VITE_API_URL: urlWithProtocol(["http", "https"]),
    VITE_WEBSOCKET_URL: urlWithProtocol(["ws", "wss"]),
});

export type Env = z.infer<typeof envSchema>;

/**
 * Parses and validates an arbitrary key/value source (import.meta.env,
 * loadEnv()'s return value, process.env, etc.) against the schema above.
 * Throws a single error listing every problem found.
 */
export function parseEnv(source: Record<string, string | undefined>): Env {
    const result = envSchema.safeParse(source);

    if (!result.success) {
        const details = result.error.issues
            .map((issue) => `  • ${issue.path.join(".") || "(root)"}: ${issue.message}`)
            .join("\n");

        const message = [
            "",
            "Invalid environment configuration:",
            "",
            details,
            "",
            "Check your .env file against .env.example.",
            "",
        ].join("\n");

        console.error(message);
        throw new Error(`Environment validation failed (${result.error.issues.length} issue(s)):\n${details}`);
    }

    return result.data;
}
