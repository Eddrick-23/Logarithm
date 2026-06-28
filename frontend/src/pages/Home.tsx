import { Box, Typography, Button, Container, Stack } from "@mui/material";
import { Link } from "@tanstack/react-router";

export default function Home() {
    return (
        <Box
            sx={{
                height: "100%",
                display: "flex",
                alignItems: "center",
            }}
        >
            <Container maxWidth="md">
                <Stack spacing={4} sx={{ alignItems: "center", textAlign: "center" }}>
                    <Typography variant="h2" sx={{ letterSpacing: "-0.02em", fontWeight: "bold" }}>
                        Welcome to Logarithm
                    </Typography>

                    <Typography variant="h6" sx={{ maxWidth: "600px" }}>
                        Streamline your debugging by sending your logs directly to Logarithm. With Logarithm, you can
                        perform real-time analysis of your application logs.
                    </Typography>

                    <Stack direction="row" spacing={3}>
                        <Button
                            component={Link}
                            to="/dashboard"
                            variant="contained"
                            sx={{ borderRadius: 2, px: 4, py: 1.5 }}
                        >
                            Dashboard
                        </Button>
                        <Button
                            component={Link}
                            to="/live-tail"
                            variant="contained"
                            sx={{ borderRadius: 2, px: 4, py: 1.5 }}
                        >
                            Live Tail
                        </Button>
                        <Button
                            component={Link}
                            to="/search"
                            variant="contained"
                            sx={{ borderRadius: 2, px: 4, py: 1.5 }}
                        >
                            Logs Table
                        </Button>
                    </Stack>
                </Stack>
            </Container>
        </Box>
    );
}
