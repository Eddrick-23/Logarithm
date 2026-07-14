import { memo } from "react";
import { severityStyles } from "../utils/severity";
import { Box, Stack, Typography } from "@mui/material";
import type { LogType } from "../types/Log";

const severityLabels: Record<LogType, string> = {
    trace: "Trace",
    debug: "Debug",
    info: "Info",
    warn: "Warning",
    error: "Error",
    fatal: "Fatal",
};

export default memo(function SeverityLegend() {
    return (
        <Stack direction="row" spacing={2} sx={{ alignItems: "center", mb: 1.5 }}>
            <Typography sx={{ fontSize: 12, color: "text.disabled" }}>Severity:</Typography>
            {(Object.keys(severityStyles) as LogType[]).map((severity) => (
                <Stack key={severity} direction="row" spacing={0.5} sx={{ alignItems: "center" }}>
                    {/* create a circle with background colour that of the severity */}
                    <Box
                        sx={{
                            width: 7,
                            height: 7,
                            borderRadius: "50%",
                            backgroundColor: severityStyles[severity].text,
                        }}
                    />
                    <Typography variant="caption" sx={{ color: "text.secondary" }}>
                        {severityLabels[severity]}
                    </Typography>
                </Stack>
            ))}
        </Stack>
    );
});
