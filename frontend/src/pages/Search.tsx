import { Box, Stack, Typography } from "@mui/material";
import LogsTable from "../components/LogsTable";

export default function Search() {
    return (
        <Box>
            <Stack spacing={2}>
                <Typography sx={{ fontSize: 13, fontWeight: 600, letterSpacing: "0.04em", color: "text.secondary" }}>
                    HISTORICAL SEARCH DASHBOARD
                </Typography>
                <LogsTable />
            </Stack>
        </Box>
    );
}
