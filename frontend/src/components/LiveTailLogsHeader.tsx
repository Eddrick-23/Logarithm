import { sectionLabel } from "../theme/tokens";
import { Stack, Typography, Box, Button } from "@mui/material";
import { memo } from "react";
import PauseIcon from "@mui/icons-material/Pause";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import { STATUS_CONFIG, type ConnectionStatus } from "../types/Connection";
import StatusIndicator from "./StatusIndicator";

interface LiveTailLogsHeaderProps {
    isPaused: boolean;
    handleResume: () => void;
    handlePause: () => void;
    connectionStatus: ConnectionStatus;
}

export default memo(function LiveTailLogsHeader({
    isPaused,
    handleResume,
    handlePause,
    connectionStatus,
}: LiveTailLogsHeaderProps) {
    const statusConfig = STATUS_CONFIG[connectionStatus];

    return (
        <Box sx={{ mb: 2.5 }}>
            <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 1 }}>
                <Stack direction="row" sx={{ alignItems: "center" }} spacing={2}>
                    <Typography sx={{ ...sectionLabel }}>Live Tail</Typography>
                    <StatusIndicator colour={statusConfig.colour} label={statusConfig.label} />
                </Stack>
                <Button
                    variant="outlined"
                    startIcon={isPaused ? <PlayArrowIcon fontSize="small" /> : <PauseIcon fontSize="small" />}
                    onClick={isPaused ? handleResume : handlePause}
                    disabled={connectionStatus !== "live"}
                    sx={{
                        color: "text.disabled",
                        borderColor: "rgba(255,255,255,0.15)",
                        textTransform: "none",
                        fontSize: 13,
                        py: 0.5,
                        minWidth: 105,
                        "&.Mui-disabled": { borderColor: "rgba(255,255,255,0.05)" },
                    }}
                >
                    {isPaused ? "Continue" : "Pause"}
                </Button>
            </Box>
            <Typography variant="body2" sx={{ color: "text.secondary" }}>
                Logs from your services appear here as they happen.
            </Typography>
        </Box>
    );
});
