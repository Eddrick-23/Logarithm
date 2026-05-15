import { Box, Stack, Typography } from "@mui/material";
import CircleIcon from "@mui/icons-material/Circle";
import { card, sectionLabel } from "../theme/tokens";

interface LatencyProps {
    name: string;
    delay: number;
}

const latencyColour = (delay: number) => {
    if (delay < 2.5) {
        return "success";
    } else if (delay < 5.0) {
        return "warning";
    } else {
        return "error";
    }
};

function LatencyRow({ name, delay }: LatencyProps) {
    return (
        <Box
            sx={{
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                py: 1,
                borderBottom: "1px solid",
                "&:last-child": { borderBottom: "none" },
            }}
        >
            <Stack direction="row" spacing={1.5} sx={{ alignItems: "center" }}>
                <Typography color={latencyColour(delay)} sx={{ display: "flex", alignItems: "center" }}>
                    <CircleIcon fontSize="small" />
                </Typography>
                <Typography>{name}</Typography>
            </Stack>

            <Typography sx={{ fontWeight: "bold" }}>{delay}ms</Typography>
        </Box>
    );
}

export default function LatencyView() {
    return (
        <>
            <Box sx={{ ...card, height: "100%" }}>
                <Typography sx={sectionLabel}>Latency</Typography>

                {/* dummy data, to replace it with for loop */}
                <LatencyRow name="p50" delay={1.2} />
                <LatencyRow name="p90" delay={3.1} />
                <LatencyRow name="p95" delay={5.4} />
            </Box>
        </>
    );
}
