import { Box, Typography } from "@mui/material";
import type { LogType } from "../types/LogType";
import { card, logRowSx, pulseSx, sectionLabel } from "../theme/tokens";

interface TailLogProps {
    time: string;
    service: string;
    messageType: LogType;
    message: string;
}

const logTypeMap: Record<LogType, string> = {
    info: "success",
    warn: "warning",
    error: "error",
};

function TailLogRow({ time, service, messageType, message }: TailLogProps) {
    return (
        <Box sx={{ ...logRowSx, fontSize: 14 }}>
            <Typography sx={{ fontFamily: "inherit", fontSize: "inherit" }}>{time}</Typography>
            <Typography color="primary" sx={{ fontFamily: "inherit", fontSize: "inherit", fontWeight: 700 }}>
                [{service}]
            </Typography>
            <Typography
                color={logTypeMap[messageType]}
                sx={{ fontFamily: "inherit", fontSize: "inherit", fontWeight: 700 }}
            >
                {messageType.toUpperCase()}
            </Typography>
            <Typography
                sx={{
                    fontFamily: "inherit",
                    fontSize: "inherit",
                    color: "text.secondary",
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                }}
            >
                {message}
            </Typography>
        </Box>
    );
}

export default function LiveTailLogs() {
    return (
        <>
            <Box sx={{ ...card, width: "100%" }}>
                <Typography sx={{ ...sectionLabel, display: "flex", alignItems: "center", gap: "6px" }}>
                    <Box sx={{ ...pulseSx, color: "success.main" }} />
                    Live Tail - All services
                </Typography>

                {/* dummy data, to replace it with for loop */}
                <TailLogRow
                    time="14:22:01.041"
                    service="api-gateway"
                    messageType="info"
                    message="request routed to /v1/orders - 200 OK in 2ms"
                />
                <TailLogRow
                    time="14:22:01.089"
                    service="auth-svc"
                    messageType="info"
                    message="token validated for user_id=8821"
                />
                <TailLogRow
                    time="14:22:01.112"
                    service="inventory"
                    messageType="warn"
                    message="stock low for item_id=4491, qty=3 remaining"
                />
                <TailLogRow
                    time="14:22:01.089"
                    service="payments"
                    messageType="info"
                    message="charge processed txn_id=TXN-08441 - $48.00"
                />
                <TailLogRow
                    time="14:22:01.089"
                    service="auth-svc"
                    messageType="error"
                    message="smtp timeout after 5000ms, retrying (2/3)"
                />
            </Box>
        </>
    );
}
