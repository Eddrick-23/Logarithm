import { Box, LinearProgress, Typography } from "@mui/material";
import { card, sectionLabel } from "../theme/tokens";

interface ServiceProps {
    service: string;
    count: number;
    colour: string;
}

// TODO: remove magic number
const MAX_SCALE = 50;

const errorData = [
    { service: "payments", count: 50, colour: "#ef5350" },
    { service: "auth-svc", count: 31, colour: "#ffa726" },
    { service: "db-proxy", count: 18, colour: "#fbc02d" },
    { service: "inventory", count: 9, colour: "#42a5f5" },
    { service: "api-gateway", count: 4, colour: "#607d8b" },
];

function ServiceErrorRow({ service, count, colour }: ServiceProps) {
    const progressValue = (count / MAX_SCALE) * 100;

    return (
        <Box
            sx={{
                display: "flex",
                alignItems: "center",
                py: 1.5,
            }}
        >
            <Typography
                sx={{
                    width: 100,
                    fontSize: "0.85rem",
                }}
            >
                {service}
            </Typography>

            {/* bar showcasing service error comparison with other errors */}
            <Box sx={{ flexGrow: 1, mx: 2 }}>
                <LinearProgress
                    variant="determinate"
                    value={progressValue}
                    sx={{
                        height: 6,
                        borderRadius: 3,
                        backgroundColor: "rgba(255, 255, 255, 0.08)",
                        "& .MuiLinearProgress-bar": {
                            backgroundColor: colour, // TODO: update with a helper function
                            borderRadius: 3,
                        },
                    }}
                />
            </Box>

            <Typography sx={{ fontWeight: "bold", width: 24, textAlign: "right" }}>{count}</Typography>
        </Box>
    );
}

export default function ServiceError() {
    return (
        <>
            <Box sx={{ ...card, height: "100%" }}>
                <Box
                    sx={{
                        display: "flex",
                        justifyContent: "space-between",
                        alignItems: "center",
                    }}
                >
                    <Typography sx={sectionLabel}>Top errors by service</Typography>
                    <Typography variant="caption" sx={{ color: "#8b949e", fontSize: "0.85rem" }}>
                        last 1h
                    </Typography>
                </Box>
                {errorData.map((item) => {
                    return (
                        <ServiceErrorRow
                            key={item.service}
                            service={item.service}
                            count={item.count}
                            colour={item.colour}
                        />
                    );
                })}
            </Box>
        </>
    );
}
