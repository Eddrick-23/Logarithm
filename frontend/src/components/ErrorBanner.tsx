import { Box, Button, Typography } from "@mui/material";
import ErrorIcon from "@mui/icons-material/Error";

interface ErrorBannerProps {
    service: string;
    handleReconnect: () => void;
}

export default function ErrorBanner({ service, handleReconnect }: ErrorBannerProps) {
    return (
        <Box
            sx={{
                display: "flex",
                alignItems: "center",
                gap: 1,
                width: "100%",
                px: 2,
                py: 1,
                border: "1px solid #7f1d1d",
                backgroundColor: "rgba(127, 29, 29, 0.15)",
                borderRadius: "6px",
                mb: 3,
            }}
        >
            <ErrorIcon sx={{ fontSize: 16, color: "#ef4444" }} />
            <Typography variant="body2" sx={{ color: "#ef4444" }}>
                Connection lost. Failed to connect to the {service}.
            </Typography>
            <Button
                size="small"
                sx={{
                    ml: "auto",
                    color: "#ef4444",
                    borderColor: "#ef4444",
                    textTransform: "none",
                    fontSize: 12,
                    "&:hover": { borderColor: "#ef4444", backgroundColor: "rgba(239, 68, 68, 0.08)" },
                }}
                variant="outlined"
                onClick={handleReconnect}
            >
                Retry
            </Button>
        </Box>
    );
}
