import { Box, Stack, Typography } from "@mui/material";
import LogsTable from "../components/LogsTable";
import SeverityLegend from "../components/SeverityLegend";

export default function Search() {
    return (
        <Box>
            <Stack spacing={2}>
                {/* Page header */}
                <Typography sx={{ fontSize: 13, fontWeight: 600, letterSpacing: "0.04em", color: "text.secondary" }}>
                    HISTORICAL SEARCH DASHBOARD
                </Typography>

                {/* severity legend colour to show mapping of severity to its associated colours */}
                <SeverityLegend />

                {/* logs table */}
                <LogsTable />
            </Stack>
        </Box>
    );
}
