import { Stack, Typography } from "@mui/material";
import PulsingCircle from "./PulsingCircle";
import { memo } from "react";
import { STATUS_CONFIG, type ConnectionStatus } from "../types/Connection";

interface StatusIndicatorProps {
    connectionStatus: ConnectionStatus;
}

export default memo(function StatusIndicator({ connectionStatus }: StatusIndicatorProps) {
    const statusConfig = STATUS_CONFIG[connectionStatus];

    return (
        <Stack direction="row" sx={{ alignItems: "center" }} spacing={1}>
            <PulsingCircle color={statusConfig.colour} />
            <Typography
                sx={{
                    fontSize: 10,
                    fontWeight: 700,
                    letterSpacing: "0.1em",
                    color: statusConfig.colour,
                }}
            >
                {statusConfig.label}
            </Typography>
        </Stack>
    );
});
