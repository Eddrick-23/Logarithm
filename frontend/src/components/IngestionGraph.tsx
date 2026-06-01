import { Typography, ListItem, Box } from "@mui/material";
import { LineChart } from "@mui/x-charts/LineChart";
import { card, sectionLabel } from "../theme/tokens";

export default function IngestionGraph() {
    return (
        <Box sx={{ ...card }}>
            <Typography sx={sectionLabel}>Ingestion Throughput - Last 60s</Typography>
            <ListItem>
                <LineChart
                    xAxis={[{ data: [1, 2, 3, 5, 8, 10] }]}
                    series={[
                        {
                            data: [2, 5.5, 2, 8.5, 1.5, 5],
                            label: "logs/s",
                            color: "#42a5f5",
                        },
                        {
                            data: [0.5, 1.2, 0.8, 2.1, 0.3, 1.4],
                            label: "errors/s",
                            color: "#f44336",
                        },
                    ]}
                    height={300}
                    grid={{ vertical: true, horizontal: true }}
                />
            </ListItem>
        </Box>
    );
}
