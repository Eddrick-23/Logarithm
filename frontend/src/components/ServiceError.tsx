import { Box, IconButton, LinearProgress, Skeleton, Tooltip, Typography } from "@mui/material";
import { card, sectionLabel } from "../theme/tokens";
import { useTopServiceErrorsStats } from "../hooks/useMetrics";
import type { TopServiceErrorsStats } from "../types/Metric";
import type { Threshold } from "../types/Threshold";
import { formatNumber, getSeverityColour, loadThresholds, saveThresholds } from "../utils/utils";
import { TOP_SERVICE_ERRORS_STATS_REFETCH_INTERVAL_MS } from "../api/metricsApi";
import { LastUpdated } from "./LastUpdated";
import { useCallback, useState } from "react";
import { ThresholdEditor } from "./ThresholdEditor";
import { ThresholdLegend } from "./ThresholdLegend";
import SettingsIcon from "@mui/icons-material/Settings";

const NUM_SERVICES = 5;

interface ServiceErrorRowProps {
    serviceName: string;
    totalErrors: number;
    errorRate: number;
    thresholds: Threshold[];
}

function ServiceErrorRow({ serviceName, totalErrors, errorRate, thresholds }: ServiceErrorRowProps) {
    return (
        <Box sx={{ display: "flex", alignItems: "center", py: 1.5 }}>
            <Tooltip title={serviceName} placement="top-start">
                <Typography
                    noWrap
                    sx={{ width: { lg: 75, xl: 120 }, fontSize: "0.85rem", flexShrink: 0, cursor: "default" }}
                >
                    {serviceName}
                </Typography>
            </Tooltip>
            <Box sx={{ flexGrow: 1, minWidth: { lg: 30, xl: 60 }, mx: 2 }}>
                <LinearProgress
                    variant="determinate"
                    value={errorRate}
                    sx={{
                        height: 6,
                        borderRadius: 3,
                        backgroundColor: "rgba(255, 255, 255, 0.08)",
                        "& .MuiLinearProgress-bar": {
                            backgroundColor: getSeverityColour(errorRate, thresholds),
                            borderRadius: 3,
                        },
                    }}
                />
            </Box>
            <Box sx={{ display: "flex", flexDirection: "column", alignItems: "flex-end", flexShrink: 0, width: 60 }}>
                <Tooltip title={totalErrors.toLocaleString()} placement="top">
                    <Typography sx={{ fontWeight: "bold", fontSize: "0.85rem" }}>
                        {formatNumber(totalErrors)}
                    </Typography>
                </Tooltip>
                <Typography sx={{ fontSize: "0.75rem", color: "text.secondary" }}>{errorRate.toFixed(2)}%</Typography>
            </Box>
        </Box>
    );
}

export default function ServiceError() {
    const { data, isLoading, isError, dataUpdatedAt } = useTopServiceErrorsStats();
    const [thresholds, setThresholds] = useState<Threshold[]>(loadThresholds);
    const [editorOpen, setEditorOpen] = useState(false);

    const handleThresholdChange = useCallback((updated: Threshold[]) => {
        // update the state and store it in local storage so that it is kept between sessions
        setThresholds(updated);
        saveThresholds(updated);
    }, []);

    return (
        <Box sx={{ ...card, height: "100%" }}>
            {/* header */}
            <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: 2 }}>
                <Typography sx={sectionLabel}>
                    Top errors by service{" "}
                    <Tooltip title="Configure thresholds" placement="top">
                        {/* button to click to configure thresholds */}
                        <IconButton
                            size="small"
                            onClick={() => setEditorOpen(true)}
                            aria-label="Configure thresholds"
                            sx={{ color: "primary.main", "&:hover": { color: "#63b4f6" } }}
                        >
                            <SettingsIcon fontSize="small" />
                        </IconButton>
                    </Tooltip>
                </Typography>
                <Typography variant="caption" sx={{ color: "primary.main", fontSize: "0.85rem", mb: 1 }}>
                    last 1h
                </Typography>
            </Box>

            {/* last updated display with refetch interval */}
            <LastUpdated timestamp={dataUpdatedAt} refreshIntervalMs={TOP_SERVICE_ERRORS_STATS_REFETCH_INTERVAL_MS} />

            {/* add a 8px gap between last updated and threshold legend */}
            <Box sx={{ mb: 1 }} />

            {/* Threshold legend */}
            {!isLoading && !isError && data && data.length > 0 && <ThresholdLegend thresholds={thresholds} />}

            {/* create loading skeleton bars to simulate loading service errors */}
            {isLoading && (
                <Box sx={{ display: "flex", flexDirection: "column", gap: 1.5, mt: 1 }}>
                    {Array.from({ length: NUM_SERVICES }).map((_, i) => (
                        <Skeleton key={i} variant="rectangular" height={36} sx={{ borderRadius: 1 }} />
                    ))}
                </Box>
            )}

            {/* no errors found in past hour */}
            {!isLoading && !isError && (!data || data.length === 0) && (
                <Box sx={{ py: 3, textAlign: "center" }}>
                    <Typography variant="body2" sx={{ color: "text.secondary" }}>
                        No errors recorded in the last hour.
                    </Typography>
                </Box>
            )}

            {/* display rows */}
            {!isLoading && !isError && data && data.length > 0 && (
                <Box>
                    {data.map((item: TopServiceErrorsStats) => (
                        <ServiceErrorRow
                            key={item.serviceName}
                            serviceName={item.serviceName}
                            totalErrors={item.totalErrors}
                            errorRate={item.errorRate}
                            thresholds={thresholds}
                        />
                    ))}
                </Box>
            )}

            {/* threshold editor  */}
            <ThresholdEditor
                open={editorOpen}
                onClose={() => setEditorOpen(false)}
                thresholds={thresholds}
                onChange={handleThresholdChange}
            />
        </Box>
    );
}
