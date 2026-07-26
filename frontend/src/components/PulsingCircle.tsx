import { pulseSx } from "../theme/tokens";
import { Box } from "@mui/material";
import { memo } from "react";

interface PulsingCircleProps {
    color: string;
}

export default memo(function PulsingCircle({ color }: PulsingCircleProps) {
    return <Box sx={{ ...pulseSx, bgcolor: color, color: color }} />;
});
