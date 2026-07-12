import { Box, IconButton, LinearProgress, Skeleton, Tooltip, Typography, Alert } from "@mui/material";
import { card, sectionLabel } from "../theme/tokens";
import { useTopServiceErrorsStats } from "../hooks/useMetrics";
import type { TopServiceErrorsStats } from "../types/Metric";
import type { Threshold } from "../types/Threshold";
import { formatNumber, getSeverityColour, loadThresholds, saveThresholds } from "../utils/utils";
import { TOP_SERVICE_ERRORS_STATS_REFETCH_INTERVAL_MS } from "../api/metricsApi";
import { LastUpdated } from "./LastUpdated";
import { memo, useCallback, useState } from "react";
import ThresholdEditor from "./ThresholdEditor";
import ThresholdLegend from "./ThresholdLegend";
import SettingsIcon from "@mui/icons-material/Settings";

const NUM_SERVICES = 5;

interface ServiceErrorRowProps {
    serviceName: string;
    totalErrors: number;
    errorRate: number;
    thresholds: Threshold[];
}

interface ServiceErrorProps {
    isLoading: boolean;
    isError: boolean;
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

export default memo(function ServiceError({ isLoading, isError }: ServiceErrorProps) {
    const { data, dataUpdatedAt } = useTopServiceErrorsStats();
    const [thresholds, setThresholds] = useState<Threshold[]>(loadThresholds);
    const [editorOpen, setEditorOpen] = useState(false);

    const handleThresholdChange = useCallback((updated: Threshold[]) => {
        // update the state and store it in local storage so that it is kept between sessions
        setThresholds(updated);
        saveThresholds(updated);
    }, []);

    const handleEditorClose = useCallback(() => {
        setEditorOpen(false);
    }, []);

    const hasData = data && data.length > 0;
    const isEmpty = !isLoading && !isError && !hasData;
    const isHardError = !isLoading && isError && !hasData;
    const isStale = !isLoading && isError && hasData;

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

            {/* Error bar */}
            {isHardError && (
                <Alert variant="outlined" severity="error">
                    Unable to load top service errors.
                </Alert>
            )}

            {/* Threshold legend */}
            {!isLoading && hasData && <ThresholdLegend thresholds={thresholds} />}

            {/* stale data: show last known data while attempting to refetch */}
            {isStale && (
                // add a gap between LastUpdatedAt and alert bar
                <Box sx={{ mt: 1.5 }}>
                    <Alert variant="outlined" severity="warning">
                        Showing last known data.
                    </Alert>
                </Box>
            )}

            {/* create loading skeleton bars to simulate loading service errors */}
            {isLoading && (
                <Box sx={{ display: "flex", flexDirection: "column", gap: 1.5, mt: 1 }}>
                    {Array.from({ length: NUM_SERVICES }).map((_, i) => (
                        <Skeleton key={i} variant="rectangular" height={36} sx={{ borderRadius: 1 }} />
                    ))}
                </Box>
            )}

            {/* no errors found in past hour */}
            {isEmpty && (
                <Box sx={{ py: 3, textAlign: "center" }}>
                    <Typography variant="body2" sx={{ color: "text.secondary" }}>
                        No errors recorded in the last hour.
                    </Typography>
                </Box>
            )}

            {/* display rows */}
            {!isLoading && hasData && (
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
                onClose={handleEditorClose}
                thresholds={thresholds}
                onChange={handleThresholdChange}
            />
        </Box>
    );
});
