import { Box, LinearProgress, Typography } from "@mui/material";
import { card, sectionLabel } from "../theme/tokens";

interface ServiceError {
    service: string;
    count: number;
}

type ServiceErrorProps = ServiceError & {
    maxCount: number;
};

const errorData: ServiceError[] = [
    { service: "payments", count: 100 },
    { service: "auth-svc", count: 79 },
    { service: "db-proxy", count: 59 },
    { service: "inventory", count: 39 },
    { service: "api-gateway", count: 19 },
];

const SEVERITY_THRESHOLDS = [
    { min: 80, colour: "#ef5350" }, // critical  — red
    { min: 60, colour: "#ffa726" }, // high      — orange
    { min: 40, colour: "#fbc02d" }, // medium    — yellow
    { min: 20, colour: "#42a5f5" }, // low       — blue
    { min: 0, colour: "#607d8b" }, // minimal   — grey
] as const;

function getSeverityColour(count: number, maxCount: number): string {
    const percentage = (count / maxCount) * 100;
    return SEVERITY_THRESHOLDS.find(({ min }) => percentage >= min)!.colour;
}

function ServiceErrorRow({ service, count, maxCount }: ServiceErrorProps) {
    const progressValue = (count / maxCount) * 100;
    return (
        <Box sx={{ display: "flex", alignItems: "center", py: 1.5, overflow: "hidden" }}>
            <Typography noWrap sx={{ minWidth: 80, maxWidth: 120, fontSize: "0.85rem", flexShrink: 0 }}>
                {service}
            </Typography>
            <Box sx={{ flexGrow: 1, flexShrink: 1, minWidth: 60, mx: 2 }}>
                <LinearProgress
                    variant="determinate"
                    value={progressValue}
                    sx={{
                        height: 6,
                        borderRadius: 3,
                        backgroundColor: "rgba(255, 255, 255, 0.08)",
                        "& .MuiLinearProgress-bar": {
                            backgroundColor: getSeverityColour(count, maxCount),
                            borderRadius: 3,
                        },
                    }}
                />
            </Box>
            <Typography sx={{ fontWeight: "bold", minWidth: 40, textAlign: "right", flexShrink: 0 }}>
                {count}
            </Typography>
        </Box>
    );
}

export default function ServiceError() {
    const maxCount = Math.max(...errorData.map((data) => data.count));

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
                            maxCount={maxCount}
                        />
                    );
                })}
            </Box>
        </>
    );
}
