import { Tooltip, Typography, type TooltipProps } from "@mui/material";
import { useRelativeTime } from "../hooks/useRelativeTime";
import { formatInterval } from "../utils/utils";

interface LastUpdatedProps {
    timestamp: number;
    refreshIntervalMs: number;
    placement?: TooltipProps["placement"];
}

export function LastUpdated({ timestamp, refreshIntervalMs, placement = "bottom-start" }: LastUpdatedProps) {
    const relativeTime = useRelativeTime(timestamp, 2);
    const intervalLabel = formatInterval(refreshIntervalMs);

    if (!relativeTime) return null;

    return (
        <Tooltip
            title={timestamp ? "Last updated at " + new Date(timestamp).toLocaleTimeString() : ""}
            placement={placement}
        >
            <Typography sx={{ fontSize: 11, color: "text.disabled", mt: 0.5 }}>
                Updated {relativeTime}
                {intervalLabel && ` · every ${intervalLabel}`}
            </Typography>
        </Tooltip>
    );
}
