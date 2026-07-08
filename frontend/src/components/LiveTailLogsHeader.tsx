import { sectionLabel } from "../theme/tokens";
import { Stack, Typography, Box, Button } from "@mui/material";
import { memo } from "react";
import PulsingCircle from "./PulsingCircle";
import PauseIcon from "@mui/icons-material/Pause";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import type { ConnectionStatus } from "../types/Connection";

interface LiveTailLogsHeaderProps {
    color: string;
    isPaused: boolean;
    handleResume: () => void;
    handlePause: () => void;
    connectionStatus: ConnectionStatus;
}

export default memo(function LiveTailLogsHeader({
    color,
    isPaused,
    handleResume,
    handlePause,
    connectionStatus,
}: LiveTailLogsHeaderProps) {
    return (
        <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 1 }}>
            <Stack direction="row" sx={{ alignItems: "center" }} spacing={1}>
                <PulsingCircle color={color} />
                <Typography sx={{ ...sectionLabel, color: color }}>Live Tail</Typography>
            </Stack>
            <Button
                variant="outlined"
                startIcon={isPaused ? <PlayArrowIcon fontSize="small" /> : <PauseIcon fontSize="small" />}
                onClick={isPaused ? handleResume : handlePause}
                disabled={connectionStatus !== "connected"}
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
    );
});
