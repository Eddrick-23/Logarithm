import * as React from "react";
import { alpha } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TablePagination from "@mui/material/TablePagination";
import TableRow from "@mui/material/TableRow";
import TableSortLabel from "@mui/material/TableSortLabel";
import Toolbar from "@mui/material/Toolbar";
import Typography from "@mui/material/Typography";
import Paper from "@mui/material/Paper";
import Checkbox from "@mui/material/Checkbox";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import DeleteIcon from "@mui/icons-material/Delete";
import FilterListIcon from "@mui/icons-material/FilterList";
import { visuallyHidden } from "@mui/utils";
import type { LogRecord } from "../types/LogRecord";
import { InputAdornment, Stack, TextField } from "@mui/material";
import SearchIcon from "@mui/icons-material/Search";

const rows: LogRecord[] = [
    {
        timestamp: new Date("2026-05-10"),
        traceId: "4bf92f3577b34da6a3ce929d0e0e4736",
        spanId: "00f067aa0ba902b7",
        severityText: "INFO",
        severityNumber: 9,
        body: "User successfully logged in",
        logAttributes: [
            { key: "userId", value: "user_123" },
            { key: "ip", value: "192.168.1.10" },
        ],
    },
    {
        timestamp: new Date(),
        traceId: "7d1f8e92ac6b4f30a77f92a51e8b2211",
        spanId: "f34c9d12abcc98ef",
        severityText: "WARN",
        severityNumber: 13,
        body: "Database response time is slow",
        logAttributes: [
            { key: "queryTimeMs", value: 1240 },
            { key: "database", value: "postgres-primary" },
        ],
    },
    {
        timestamp: new Date(),
        traceId: "ab23ff11cc88aa77dd44ee99bb00cc11",
        spanId: "c8e1a7d55f00aa12",
        severityText: "ERROR",
        severityNumber: 17,
        body: "Failed to process payment",
        logAttributes: [
            { key: "paymentId", value: "pay_98765" },
            { key: "reason", value: "Insufficient funds" },
        ],
    },
    {
        timestamp: new Date(),
        traceId: "1aa22bb33cc44dd55ee66ff77889900",
        spanId: "eeddccbbaa998877",
        severityText: "DEBUG",
        severityNumber: 5,
        body: "Fetching user profile cache",
        logAttributes: [
            { key: "cacheHit", value: true },
            { key: "region", value: "ap-southeast-1" },
        ],
    },
    {
        timestamp: new Date(),
        traceId: "2cc33dd44ee55ff6677889900aabb11",
        spanId: "1122334455667788",
        severityText: "INFO",
        severityNumber: 9,
        body: "User profile updated",
        logAttributes: [{ key: "userId", value: "user_456" }],
    },
    {
        timestamp: new Date(),
        traceId: "3dd44ee55ff6677889900aabb112233",
        spanId: "2233445566778899",
        severityText: "WARN",
        severityNumber: 13,
        body: "Redis memory usage high",
        logAttributes: [{ key: "memoryPercent", value: 91 }],
    },
    {
        timestamp: new Date(),
        traceId: "4ee55ff6677889900aabb1122334455",
        spanId: "3344556677889900",
        severityText: "ERROR",
        severityNumber: 17,
        body: "Kafka broker disconnected",
        logAttributes: [{ key: "broker", value: "kafka-1" }],
    },
    {
        timestamp: new Date(),
        traceId: "5ff6677889900aabb11223344556677",
        spanId: "4455667788990011",
        severityText: "DEBUG",
        severityNumber: 5,
        body: "JWT token validation started",
        logAttributes: [{ key: "service", value: "auth-service" }],
    },
    {
        timestamp: new Date(),
        traceId: "6677889900aabb112233445566778899",
        spanId: "5566778899001122",
        severityText: "INFO",
        severityNumber: 9,
        body: "Email sent successfully",
        logAttributes: [{ key: "recipient", value: "user@example.com" }],
    },
    {
        timestamp: new Date(),
        traceId: "77889900aabb112233445566778899aa",
        spanId: "6677889900112233",
        severityText: "WARN",
        severityNumber: 13,
        body: "API rate limit approaching",
        logAttributes: [{ key: "requestsPerMinute", value: 950 }],
    },
    {
        timestamp: new Date(),
        traceId: "889900aabb112233445566778899aabb",
        spanId: "7788990011223344",
        severityText: "ERROR",
        severityNumber: 17,
        body: "Unable to connect to MongoDB",
        logAttributes: [{ key: "cluster", value: "mongo-prod" }],
    },
    {
        timestamp: new Date(),
        traceId: "9900aabb112233445566778899aabbcc",
        spanId: "8899001122334455",
        severityText: "DEBUG",
        severityNumber: 5,
        body: "Cache invalidation triggered",
        logAttributes: [{ key: "cacheKey", value: "user-profile" }],
    },
    {
        timestamp: new Date(),
        traceId: "aabb112233445566778899aabbccddeeff",
        spanId: "9900112233445566",
        severityText: "INFO",
        severityNumber: 9,
        body: "File uploaded successfully",
        logAttributes: [{ key: "fileName", value: "invoice.pdf" }],
    },
    {
        timestamp: new Date(),
        traceId: "bbcc2233445566778899aabbccddeeff",
        spanId: "0011223344556677",
        severityText: "WARN",
        severityNumber: 13,
        body: "CPU usage spike detected",
        logAttributes: [{ key: "cpuPercent", value: 88 }],
    },
    {
        timestamp: new Date(),
        traceId: "ccdd33445566778899aabbccddeeff00",
        spanId: "1122446688001133",
        severityText: "ERROR",
        severityNumber: 17,
        body: "Stripe payment timeout",
        logAttributes: [{ key: "timeoutMs", value: 30000 }],
    },
    {
        timestamp: new Date(),
        traceId: "ddee445566778899aabbccddeeff0011",
        spanId: "2233557799112244",
        severityText: "DEBUG",
        severityNumber: 5,
        body: "Metrics aggregation completed",
        logAttributes: [{ key: "durationMs", value: 210 }],
    },
    {
        timestamp: new Date(),
        traceId: "eeff5566778899aabbccddeeff001122",
        spanId: "3344668800223355",
        severityText: "INFO",
        severityNumber: 9,
        body: "Service health check passed",
        logAttributes: [{ key: "service", value: "gateway" }],
    },
    {
        timestamp: new Date(),
        traceId: "ff0066778899aabbccddeeff00112233",
        spanId: "4455779911334466",
        severityText: "WARN",
        severityNumber: 13,
        body: "Disk usage exceeded threshold",
        logAttributes: [{ key: "diskPercent", value: 94 }],
    },
    {
        timestamp: new Date(),
        traceId: "0011778899aabbccddeeff0011223344",
        spanId: "5566880022445577",
        severityText: "ERROR",
        severityNumber: 17,
        body: "Failed to refresh OAuth token",
        logAttributes: [{ key: "provider", value: "Google" }],
    },
    {
        timestamp: new Date(),
        traceId: "11228899aabbccddeeff001122334455",
        spanId: "6677991133556688",
        severityText: "DEBUG",
        severityNumber: 5,
        body: "WebSocket heartbeat received",
        logAttributes: [{ key: "connectionId", value: "ws_789" }],
    },
];

function descendingComparator<T>(a: T, b: T, orderBy: keyof T) {
    const first = a[orderBy];
    const second = b[orderBy];

    if (first instanceof Date && second instanceof Date) {
        return second.getTime() - first.getTime();
    }

    if (second < first) {
        return -1;
    }

    if (second > first) {
        return 1;
    }

    return 0;
}

type Order = "asc" | "desc";

function getComparator<Key extends keyof LogRecord>(
    order: Order,
    orderBy: Key,
): (a: LogRecord, b: LogRecord) => number {
    return order === "desc"
        ? (a, b) => descendingComparator(a, b, orderBy)
        : (a, b) => -descendingComparator(a, b, orderBy);
}

interface HeadCell {
    id: keyof LogRecord;
    label: string;
}

const headCells: readonly HeadCell[] = [
    {
        id: "traceId",
        label: "Trace ID",
    },
    {
        id: "spanId",
        label: "Span ID",
    },
    {
        id: "severityText",
        label: "Severity Text",
    },
    {
        id: "severityNumber",
        label: "Severity Number",
    },
    {
        id: "body",
        label: "Body",
    },
    {
        id: "timestamp",
        label: "Time",
    },
];

interface EnhancedTableProps {
    numSelected: number;
    onRequestSort: (event: React.MouseEvent<unknown>, property: keyof LogRecord) => void;
    onSelectAllClick: (event: React.ChangeEvent<HTMLInputElement>) => void;
    order: Order;
    orderBy: string;
    rowCount: number;
}

function EnhancedTableHead(props: EnhancedTableProps) {
    const { onSelectAllClick, order, orderBy, numSelected, rowCount, onRequestSort } = props;
    const createSortHandler = (property: keyof LogRecord) => (event: React.MouseEvent<unknown>) => {
        onRequestSort(event, property);
    };

    return (
        <TableHead>
            <TableRow>
                <TableCell padding="checkbox">
                    <Checkbox
                        color="primary"
                        indeterminate={numSelected > 0 && numSelected < rowCount}
                        checked={rowCount > 0 && numSelected === rowCount}
                        onChange={onSelectAllClick}
                        slotProps={{
                            input: { "aria-label": "select all desserts" },
                        }}
                    />
                </TableCell>
                {headCells.map((headCell) => (
                    <TableCell key={headCell.id} sortDirection={orderBy === headCell.id ? order : false}>
                        <TableSortLabel
                            active={orderBy === headCell.id}
                            direction={orderBy === headCell.id ? order : "asc"}
                            onClick={createSortHandler(headCell.id)}
                        >
                            {headCell.label}
                            {orderBy === headCell.id ? (
                                <Box component="span" sx={visuallyHidden}>
                                    {order === "desc" ? "sorted descending" : "sorted ascending"}
                                </Box>
                            ) : null}
                        </TableSortLabel>
                    </TableCell>
                ))}
            </TableRow>
        </TableHead>
    );
}
interface EnhancedTableToolbarProps {
    numSelected: number;
}
function EnhancedTableToolbar(props: EnhancedTableToolbarProps) {
    const { numSelected } = props;
    const [searchQuery, setSearchQuery] = React.useState<string>("");

    return (
        <Toolbar
            sx={[
                {
                    pl: { sm: 2 },
                    pr: { xs: 1, sm: 1 },
                },
                numSelected > 0 && {
                    bgcolor: (theme) => alpha(theme.palette.primary.main, theme.palette.action.activatedOpacity),
                },
            ]}
        >
            {numSelected > 0 ? (
                <Typography
                    variant="subtitle1"
                    component="div"
                    sx={{
                        color: "inherit",
                        flex: "1 1 100%",
                    }}
                >
                    {numSelected} selected
                </Typography>
            ) : (
                <Typography sx={{ flex: "1 1 100%" }} variant="h6" id="tableTitle" component="div">
                    Logs
                </Typography>
            )}
            <Stack direction="row" spacing={2}>
                <TextField
                    placeholder="Search"
                    slotProps={{
                        input: {
                            startAdornment: (
                                <InputAdornment position="start">
                                    <SearchIcon />
                                </InputAdornment>
                            ),
                        },
                    }}
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    variant="standard"
                />
                {numSelected > 0 ? (
                    <Tooltip title="Delete">
                        <IconButton>
                            <DeleteIcon />
                        </IconButton>
                    </Tooltip>
                ) : (
                    <Tooltip title="Filter list">
                        <IconButton>
                            <FilterListIcon />
                        </IconButton>
                    </Tooltip>
                )}
            </Stack>
        </Toolbar>
    );
}
export default function EnhancedTable() {
    const [order, setOrder] = React.useState<Order>("asc");
    const [orderBy, setOrderBy] = React.useState<keyof LogRecord>("traceId");
    const [selected, setSelected] = React.useState<readonly string[]>([]);
    const [page, setPage] = React.useState(0);
    const [rowsPerPage, setRowsPerPage] = React.useState(5);

    const handleRequestSort = (event: React.MouseEvent<unknown>, property: keyof LogRecord) => {
        const isAsc = orderBy === property && order === "asc";
        setOrder(isAsc ? "desc" : "asc");
        setOrderBy(property);
    };

    const handleSelectAllClick = (event: React.ChangeEvent<HTMLInputElement>) => {
        if (event.target.checked) {
            const newSelected = rows.map((n) => n.traceId);
            setSelected(newSelected);
            return;
        }
        setSelected([]);
    };

    const handleClick = (event: React.MouseEvent<unknown>, id: string) => {
        const selectedIndex = selected.indexOf(id);
        let newSelected: readonly string[] = [];

        if (selectedIndex === -1) {
            newSelected = newSelected.concat(selected, id);
        } else if (selectedIndex === 0) {
            newSelected = newSelected.concat(selected.slice(1));
        } else if (selectedIndex === selected.length - 1) {
            newSelected = newSelected.concat(selected.slice(0, -1));
        } else if (selectedIndex > 0) {
            newSelected = newSelected.concat(selected.slice(0, selectedIndex), selected.slice(selectedIndex + 1));
        }
        setSelected(newSelected);
    };

    const handleChangePage = (event: unknown, newPage: number) => {
        setPage(newPage);
    };

    const handleChangeRowsPerPage = (event: React.ChangeEvent<HTMLInputElement>) => {
        setRowsPerPage(parseInt(event.target.value, 10));
        setPage(0);
    };

    // Avoid a layout jump when reaching the last page with empty rows.
    const emptyRows = page > 0 ? Math.max(0, (1 + page) * rowsPerPage - rows.length) : 0;

    const visibleRows = React.useMemo(
        () => [...rows].sort(getComparator(order, orderBy)).slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage),
        [order, orderBy, page, rowsPerPage],
    );

    return (
        <Box sx={{ width: "100%" }}>
            <Paper sx={{ width: "100%", mb: 2 }}>
                <EnhancedTableToolbar numSelected={selected.length} />
                <TableContainer>
                    <Table sx={{ minWidth: 750 }} aria-labelledby="tableTitle">
                        <EnhancedTableHead
                            numSelected={selected.length}
                            order={order}
                            orderBy={orderBy}
                            onSelectAllClick={handleSelectAllClick}
                            onRequestSort={handleRequestSort}
                            rowCount={rows.length}
                        />
                        <TableBody>
                            {visibleRows.map((row, index) => {
                                const isItemSelected = selected.includes(row.traceId);
                                const labelId = `enhanced-table-checkbox-${index}`;

                                return (
                                    <TableRow
                                        hover
                                        onClick={(event) => handleClick(event, row.traceId)}
                                        role="checkbox"
                                        aria-checked={isItemSelected}
                                        tabIndex={-1}
                                        key={row.traceId}
                                        selected={isItemSelected}
                                        sx={{ cursor: "pointer" }}
                                    >
                                        <TableCell padding="checkbox">
                                            <Checkbox
                                                color="primary"
                                                checked={isItemSelected}
                                                slotProps={{
                                                    input: { "aria-labelledby": labelId },
                                                }}
                                            />
                                        </TableCell>
                                        <TableCell component="th" id={labelId} scope="row">
                                            {row.traceId}
                                        </TableCell>
                                        <TableCell>{row.spanId}</TableCell>
                                        <TableCell>{row.severityText}</TableCell>
                                        <TableCell>{row.severityNumber}</TableCell>
                                        <TableCell>{row.body}</TableCell>
                                        <TableCell>{row.timestamp.toLocaleString()}</TableCell>
                                    </TableRow>
                                );
                            })}
                            {emptyRows > 0 && (
                                <TableRow>
                                    <TableCell colSpan={6} />
                                </TableRow>
                            )}
                        </TableBody>
                    </Table>
                </TableContainer>
                <TablePagination
                    rowsPerPageOptions={[5, 10, 25]}
                    component="div"
                    count={rows.length}
                    rowsPerPage={rowsPerPage}
                    page={page}
                    onPageChange={handleChangePage}
                    onRowsPerPageChange={handleChangeRowsPerPage}
                />
            </Paper>
        </Box>
    );
}
