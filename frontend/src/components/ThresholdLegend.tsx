import { memo, useMemo } from "react";
import type { Threshold } from "../types/Threshold";
import { Box, Typography } from "@mui/material";

interface ThresholdLegendProps {
    thresholds: Threshold[];
}

export default memo(function ThresholdLegend({ thresholds }: ThresholdLegendProps) {
    const sorted = useMemo(() => [...thresholds].sort((a, b) => b.min - a.min), [thresholds]);
    return (
        <Box sx={{ display: "flex", flexWrap: "wrap", gap: 1, mb: 1.5 }}>
            {sorted.map((t) => (
                <Box key={t.min} sx={{ display: "flex", alignItems: "center", gap: 0.5 }}>
                    {/* circle */}
                    <Box
                        sx={{
                            width: 8,
                            height: 8,
                            borderRadius: "50%",
                            backgroundColor: t.colour,
                            flexShrink: 0,
                        }}
                    />

                    {/* label */}
                    <Typography sx={{ fontSize: "0.7rem", color: "text.secondary" }}>
                        {t.label} ≥{t.min}%
                    </Typography>
                </Box>
            ))}
        </Box>
    );
});
