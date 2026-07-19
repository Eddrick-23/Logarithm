import { Box, Typography } from "@mui/material";
import { severityStyles } from "../utils/severity";
import type { LogType } from "../types/Log";
import { memo } from "react";

interface SeverityPillProps {
    severity: LogType;
}

export default memo(function SeverityPill({ severity }: SeverityPillProps) {
    return (
        <Box
            sx={{
                display: "inline-block",
                backgroundColor: severityStyles[severity].bg,
                borderRadius: 1,
                px: 1,
                py: 0.25,
            }}
        >
            <Typography
                sx={{
                    fontSize: 11,
                    fontWeight: "bold",
                    color: severityStyles[severity].text,
                }}
            >
                {severity.toUpperCase()}
            </Typography>
        </Box>
    );
});
