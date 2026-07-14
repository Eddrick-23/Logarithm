import { Stack, Typography } from "@mui/material";
import PulsingCircle from "./PulsingCircle";
import { memo } from "react";

interface StatusIndicatorProps {
    colour: string;
    label: string;
}

export default memo(function StatusIndicator({ colour, label }: StatusIndicatorProps) {
    return (
        <Stack direction="row" sx={{ alignItems: "center" }} spacing={1}>
            <PulsingCircle color={colour} />
            <Typography
                sx={{
                    fontSize: 10,
                    fontWeight: 700,
                    letterSpacing: "0.1em",
                    color: colour,
                }}
            >
                {label}
            </Typography>
        </Stack>
    );
});
