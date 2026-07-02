import {
    Box,
    Button,
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle,
    IconButton,
    TextField,
    Tooltip,
    Typography,
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import CloseIcon from "@mui/icons-material/Close";
import RestartAltIcon from "@mui/icons-material/RestartAlt";
import { memo, useEffect, useState } from "react";
import { MuiColorInput } from "mui-color-input";
import { type Threshold } from "../types/Threshold";
import { DEFAULT_THRESHOLDS, HEX_COLOUR_REGEX } from "../utils/utils";

interface ThresholdEditorProps {
    open: boolean;
    onClose: () => void;
    thresholds: Threshold[];
    onChange: (updated: Threshold[]) => void;
}

const MAX_BANDS = 7;

export const ThresholdEditor = memo(function ThresholdEditor({
    open,
    onClose,
    thresholds,
    onChange,
}: ThresholdEditorProps) {
    const [draft, setDraft] = useState<Threshold[]>(thresholds);
    const [errors, setErrors] = useState<Record<number, string>>({});

    useEffect(() => {
        if (open) {
            setDraft([...thresholds].sort((a, b) => b.min - a.min));
            setErrors({});
        }
    }, [open, thresholds]);

    const validate = (rows: Threshold[]): Record<number, string> => {
        const errs: Record<number, string> = {};
        const mins = rows.map((r) => r.min);

        // thresholds should be from 0 to 100 inclusive, have no duplicates and have labels
        rows.forEach((row, i) => {
            if (row.min < 0 || row.min > 100) {
                errs[i] = "Must be 0–100";
            } else if (mins.filter((m) => m === row.min).length > 1) {
                errs[i] = "Duplicate value";
            } else if (!row.label.trim()) {
                errs[i] = "Label required";
            } else if (!HEX_COLOUR_REGEX.test(row.colour)) {
                errs[i] = "Invalid colour";
            }
        });

        if (!rows.some((r) => r.min === 0)) {
            const lowestIndex = rows.reduce((lowest, r, i) => (r.min < rows[lowest].min ? i : lowest), 0);
            errs[lowestIndex] = (errs[lowestIndex] ? errs[lowestIndex] + "; " : "") + "One band must start at 0";
        }

        return errs;
    };

    const updateRow = (index: number, field: keyof Threshold, value: string | number) => {
        const updated = draft.map((row, i) =>
            i === index ? { ...row, [field]: field === "min" ? Number(value) : value } : row,
        );
        setDraft(updated);
        setErrors(validate(updated));
    };

    const addRow = () => {
        if (draft.length >= MAX_BANDS) return;
        const maxMin = Math.max(...draft.map((r) => r.min));
        const newMin = Math.min(maxMin + 10, 99);
        const updated = [...draft, { min: newMin, colour: "#90a4ae", label: "New band" }];
        setDraft(updated);
        setErrors(validate(updated));
    };

    const removeRow = (index: number) => {
        const updated = draft.filter((_, i) => i !== index);
        setDraft(updated);
        setErrors(validate(updated));
    };

    const handleSave = () => {
        const errs = validate(draft);
        if (Object.keys(errs).length > 0) {
            setErrors(errs);
            return;
        }
        const sorted = [...draft].sort((a, b) => b.min - a.min);
        onChange(sorted);
        onClose();
    };

    const handleReset = () => {
        setDraft(DEFAULT_THRESHOLDS);
        setErrors({});
    };

    const hasErrors = Object.keys(errors).length > 0;
    const zeroCount = draft.filter((r) => r.min === 0).length;
    const isFloor = (row: Threshold) => row.min === 0 && zeroCount === 1;

    return (
        <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
            <DialogTitle sx={{ display: "flex", alignItems: "center", justifyContent: "space-between", pb: 1 }}>
                <Typography sx={{ fontWeight: 600, fontSize: "1rem" }}>Error rate thresholds</Typography>
                <IconButton size="small" onClick={onClose} aria-label="Close">
                    <CloseIcon fontSize="small" />
                </IconButton>
            </DialogTitle>

            <DialogContent sx={{ pt: 0 }}>
                <Typography variant="caption" sx={{ color: "text.secondary", display: "block", mb: 2 }}>
                    Bands are matched top-down. The band with the highest min value that the error rate meets is
                    applied. A band starting at 0 is required as the catch-all floor. Maximum of {MAX_BANDS} bands.
                </Typography>

                {/* Column headers */}
                <Box sx={{ display: "grid", gridTemplateColumns: "80px 1fr 160px 36px", gap: 1, mb: 0.5, px: 0.5 }}>
                    {["Min %", "Label", "Colour", ""].map((header) => (
                        <Typography
                            key={header}
                            variant="caption"
                            sx={{
                                color: "text.secondary",
                                fontSize: "0.7rem",
                                textTransform: "uppercase",
                                letterSpacing: "0.05em",
                            }}
                        >
                            {header}
                        </Typography>
                    ))}
                </Box>

                {draft.map((row, i) => {
                    const error = errors[i];
                    const floor = isFloor(row);

                    return (
                        <Box
                            key={i}
                            sx={{
                                display: "grid",
                                gridTemplateColumns: "80px 1fr 160px 36px",
                                gap: 1,
                                mb: 1,
                                alignItems: "center",
                            }}
                        >
                            {/* Min value */}
                            <TextField
                                type="number"
                                value={row.min}
                                disabled={floor}
                                onChange={(e) => updateRow(i, "min", e.target.value)}
                                error={!!error}
                                size="small"
                                slotProps={{ htmlInput: { min: 0, max: 100 } }}
                                sx={{ "& .MuiInputBase-root": { fontSize: "0.85rem" } }}
                            />

                            {/* Label */}
                            <TextField
                                value={row.label}
                                onChange={(e) => updateRow(i, "label", e.target.value)}
                                placeholder="Label"
                                error={!!error}
                                size="small"
                                sx={{ "& .MuiInputBase-root": { fontSize: "0.85rem" } }}
                            />

                            {/* Colour picker */}
                            <MuiColorInput
                                value={row.colour}
                                onChange={(value) => updateRow(i, "colour", value)}
                                format="hex"
                                size="small"
                                sx={{ "& .MuiInputBase-root": { fontSize: "0.85rem" } }}
                            />

                            {/* Delete */}
                            <Tooltip
                                title={floor ? "The floor band (min 0) cannot be removed" : "Remove band"}
                                placement="top"
                            >
                                <span>
                                    <IconButton
                                        size="small"
                                        disabled={floor}
                                        onClick={() => removeRow(i)}
                                        aria-label="Remove band"
                                        sx={{
                                            color: floor ? "transparent" : "text.secondary",
                                            "&:hover": { color: "#f44336" },
                                        }}
                                    >
                                        <DeleteIcon fontSize="small" />
                                    </IconButton>
                                </span>
                            </Tooltip>

                            {/* Inline error */}
                            {error && (
                                <Typography
                                    variant="caption"
                                    sx={{ color: "#f44336", fontSize: "0.7rem", gridColumn: "1 / -1", ml: 0.5 }}
                                >
                                    {error}
                                </Typography>
                            )}
                        </Box>
                    );
                })}

                {/* Add band */}
                <Button
                    onClick={addRow}
                    disabled={draft.length >= MAX_BANDS}
                    startIcon={<AddIcon />}
                    variant="outlined"
                    size="small"
                    sx={{
                        mt: 1,
                        mb: 2,
                        borderStyle: "dashed",
                        color: "text.secondary",
                        borderColor: "rgba(255,255,255,0.18)",
                        "&:hover": { borderStyle: "dashed", borderColor: "rgba(255,255,255,0.35)" },
                    }}
                >
                    Add band
                </Button>
            </DialogContent>

            <DialogActions
                sx={{
                    borderTop: "1px solid rgba(255,255,255,0.08)",
                    justifyContent: "space-between",
                    px: 3,
                    py: 1.5,
                }}
            >
                <Button
                    onClick={handleReset}
                    startIcon={<RestartAltIcon />}
                    size="small"
                    sx={{ color: "text.secondary" }}
                >
                    Reset to defaults
                </Button>

                <Box sx={{ display: "flex", gap: 1 }}>
                    <Button onClick={onClose} size="small" variant="outlined">
                        Cancel
                    </Button>
                    <Tooltip title={hasErrors ? "Fix errors before saving" : ""} placement="top">
                        <span>
                            <Button
                                onClick={handleSave}
                                disabled={hasErrors}
                                size="small"
                                variant="contained"
                                sx={{
                                    background: hasErrors ? undefined : "#238636",
                                    "&:hover": { background: "#2ea043" },
                                }}
                            >
                                Save
                            </Button>
                        </span>
                    </Tooltip>
                </Box>
            </DialogActions>
        </Dialog>
    );
});
