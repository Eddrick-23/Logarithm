import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { DEFAULT_THRESHOLDS } from "../../utils/utils";
import type { Threshold } from "../../types/Threshold";
import { ThresholdEditor } from "../../components/ThresholdEditor";

// Stub MUI colour input since it does not work well in jsdom
vi.mock("mui-color-input", () => ({
    MuiColorInput: ({ value, onChange }: { value: string; onChange: (v: string) => void }) => (
        <input aria-label="colour" value={value} onChange={(e) => onChange(e.target.value)} />
    ),
}));

const BASE_THRESHOLDS: Threshold[] = [
    { min: 80, colour: "#ef5350", label: "Critical" },
    { min: 60, colour: "#ffa726", label: "High" },
    { min: 40, colour: "#fbc02d", label: "Medium" },
    { min: 20, colour: "#42a5f5", label: "Low" },
    { min: 0, colour: "#607d8b", label: "Minimal" },
];

function renderEditor(props?: Partial<React.ComponentProps<typeof ThresholdEditor>>) {
    const onClose = vi.fn();
    const onChange = vi.fn();

    render(
        <ThresholdEditor open={true} onClose={onClose} thresholds={BASE_THRESHOLDS} onChange={onChange} {...props} />,
    );

    return { onClose, onChange };
}

describe("ThresholdEditor — rendering", () => {
    it("renders the dialog when open is true", () => {
        renderEditor();
        expect(screen.getByText("Error rate thresholds")).toBeInTheDocument();
    });

    it("does not render dialog content when open is false", () => {
        renderEditor({ open: false });
        expect(screen.queryByText("Error rate thresholds")).not.toBeInTheDocument();
    });

    it("renders a row for each threshold sorted descending by min", () => {
        renderEditor();
        const inputs = screen.getAllByRole("spinbutton"); // percentage number inputs
        const values = inputs.map((i) => Number((i as HTMLInputElement).value));
        expect(values).toEqual([80, 60, 40, 20, 0]);
    });

    it("renders label inputs with correct values", () => {
        renderEditor();
        expect(screen.getByDisplayValue("Critical")).toBeInTheDocument();
        expect(screen.getByDisplayValue("Minimal")).toBeInTheDocument();
    });

    it("disables the min input for the floor band (min === 0)", () => {
        renderEditor();
        const inputs = screen.getAllByRole("spinbutton") as HTMLInputElement[];
        const floorInput = inputs.find((i) => i.value === "0");
        expect(floorInput).toBeDisabled();
    });

    it("disables the delete button for the floor band", () => {
        renderEditor();
        const deleteButtons = screen.getAllByRole("button", { name: /remove band/i });
        // floor band delete should be disabled — all others enabled
        const disabledDeletes = deleteButtons.filter((b) => b.hasAttribute("disabled"));
        expect(disabledDeletes).toHaveLength(1);
    });
});

describe("ThresholdEditor — adding bands", () => {
    it("adds a new row when Add band is clicked", async () => {
        const user = userEvent.setup();
        renderEditor();

        const before = screen.getAllByRole("spinbutton").length;
        await user.click(screen.getByRole("button", { name: /add band/i }));
        expect(screen.getAllByRole("spinbutton")).toHaveLength(before + 1);
    });

    it("disables Add band when MAX_BANDS (7) is reached", async () => {
        const user = userEvent.setup();
        renderEditor();

        const addBtn = screen.getByRole("button", { name: /add band/i });

        // add 2 more to reach 7
        await user.click(addBtn);
        await user.click(addBtn);

        // this means that you cannot click the add button after the limit
        expect(addBtn).toBeDisabled();
    });
});

describe("ThresholdEditor — removing bands", () => {
    it("removes a row when its delete button is clicked", async () => {
        const user = userEvent.setup();
        renderEditor();

        const before = screen.getAllByRole("spinbutton").length;
        const deleteButtons = screen.getAllByRole("button", { name: /remove band/i });
        const enabledDelete = deleteButtons.find((b) => !b.hasAttribute("disabled"))!;
        await user.click(enabledDelete);

        expect(screen.getAllByRole("spinbutton")).toHaveLength(before - 1);
    });

    it("allows deleting either band when two rows both have min 0", async () => {
        const thresholdsWithTwoZeros: Threshold[] = [
            { min: 0, colour: "#607d8b", label: "Minimal" },
            { min: 0, colour: "#42a5f5", label: "Also zero" },
        ];
        renderEditor({ thresholds: thresholdsWithTwoZeros });

        const deleteButtons = screen.getAllByRole("button", { name: /remove band/i });
        const disabledDeletes = deleteButtons.filter((b) => b.hasAttribute("disabled"));
        expect(disabledDeletes).toHaveLength(0);
    });
});

describe("ThresholdEditor — editing rows", () => {
    it("updates a label when the user types in the label field", async () => {
        const user = userEvent.setup();
        renderEditor();

        const labelInput = screen.getByDisplayValue("High");
        await user.clear(labelInput);
        await user.type(labelInput, "Elevated");

        expect(screen.getByDisplayValue("Elevated")).toBeInTheDocument();
    });

    it("updates the min value when the user changes the number input", async () => {
        const user = userEvent.setup();
        renderEditor();

        const inputs = screen.getAllByRole("spinbutton") as HTMLInputElement[];
        const highInput = inputs.find((i) => i.value === "60")!;
        await user.clear(highInput);
        await user.type(highInput, "55");

        expect(screen.getByDisplayValue("55")).toBeInTheDocument();
    });
});

describe("ThresholdEditor — validation", () => {
    it("shows an error when a label is cleared", async () => {
        const user = userEvent.setup();
        renderEditor();

        const labelInput = screen.getByDisplayValue("High");
        await user.clear(labelInput);

        expect(screen.getByText("Label required")).toBeInTheDocument();
    });

    it("shows an error for duplicate min values", async () => {
        const user = userEvent.setup();
        renderEditor();

        const inputs = screen.getAllByRole("spinbutton") as HTMLInputElement[];
        const highInput = inputs.find((i) => i.value === "60")!;
        await user.clear(highInput);
        await user.type(highInput, "80"); // duplicate of Critical

        // there should exist 2 elements with duplicate values
        const duplicateErrors = screen.getAllByText("Duplicate value");
        expect(duplicateErrors).toHaveLength(2);
    });

    it("blocks save and shows errors when there are validation issues", async () => {
        const user = userEvent.setup();
        renderEditor();

        const labelInput = screen.getByDisplayValue("High");
        await user.clear(labelInput);

        const saveBtn = screen.getByRole("button", { name: /save/i });
        expect(saveBtn).toBeDisabled();
    });

    it("shows a range error when min is set above 100", async () => {
        const user = userEvent.setup();
        renderEditor();

        const inputs = screen.getAllByRole("spinbutton") as HTMLInputElement[];
        const criticalInput = inputs.find((i) => i.value === "80")!;
        await user.clear(criticalInput);
        await user.type(criticalInput, "101");

        expect(screen.getByText("Must be 0–100")).toBeInTheDocument();
    });
});

describe("ThresholdEditor — save", () => {
    it("calls onChange with thresholds sorted descending and calls onClose", async () => {
        const user = userEvent.setup();
        const { onChange, onClose } = renderEditor();

        await user.click(screen.getByRole("button", { name: /save/i }));

        expect(onChange).toHaveBeenCalledOnce();
        const saved: Threshold[] = onChange.mock.calls[0][0];
        const mins = saved.map((t) => t.min);
        expect(mins).toEqual([...mins].sort((a, b) => b - a));
        expect(onClose).toHaveBeenCalledOnce();
    });

    it("does not call onChange when there are validation errors", async () => {
        const user = userEvent.setup();
        const { onChange } = renderEditor();

        await user.clear(screen.getByDisplayValue("High"));

        const saveBtn = screen.getByRole("button", { name: /save/i });
        expect(saveBtn).toBeDisabled();
        expect(onChange).not.toHaveBeenCalled();
    });
});

describe("ThresholdEditor — cancel", () => {
    it("calls onClose when Cancel is clicked", async () => {
        const user = userEvent.setup();
        const { onClose } = renderEditor();

        await user.click(screen.getByRole("button", { name: /cancel/i }));

        expect(onClose).toHaveBeenCalledOnce();
    });

    it("does not call onChange when Cancel is clicked", async () => {
        const user = userEvent.setup();
        const { onChange } = renderEditor();

        await user.click(screen.getByRole("button", { name: /cancel/i }));

        expect(onChange).not.toHaveBeenCalled();
    });

    it("calls onClose when the X button is clicked", async () => {
        const user = userEvent.setup();
        const { onClose } = renderEditor();

        await user.click(screen.getByRole("button", { name: /close/i }));

        expect(onClose).toHaveBeenCalledOnce();
    });
});

describe("ThresholdEditor — reset", () => {
    it("restores DEFAULT_THRESHOLDS when Reset is clicked", async () => {
        const user = userEvent.setup();
        renderEditor();

        // make a change first
        await user.clear(screen.getByDisplayValue("High"));
        await user.type(screen.getByDisplayValue(""), "Changed");

        await user.click(screen.getByRole("button", { name: /reset to defaults/i }));

        DEFAULT_THRESHOLDS.forEach((t) => {
            expect(screen.getByDisplayValue(t.label)).toBeInTheDocument();
        });
    });

    it("clears validation errors after reset", async () => {
        const user = userEvent.setup();
        renderEditor();

        await user.clear(screen.getByDisplayValue("High"));
        expect(screen.getByText("Label required")).toBeInTheDocument();

        await user.click(screen.getByRole("button", { name: /reset to defaults/i }));

        expect(screen.queryByText("Label required")).not.toBeInTheDocument();
    });

    it("does not save after reset without an explicit Save click", async () => {
        const user = userEvent.setup();
        const { onChange } = renderEditor();

        await user.click(screen.getByRole("button", { name: /reset to defaults/i }));

        expect(onChange).not.toHaveBeenCalled();
    });
});

describe("ThresholdEditor — re-open behaviour", () => {
    it("resets draft to current thresholds when dialog is reopened", async () => {
        const user = userEvent.setup();
        const { rerender } = render(
            <ThresholdEditor open={true} onClose={vi.fn()} thresholds={BASE_THRESHOLDS} onChange={vi.fn()} />,
        );

        // make an unsaved edit
        await user.clear(screen.getByDisplayValue("High"));
        await user.type(screen.getByDisplayValue(""), "Edited");

        // close and reopen
        rerender(<ThresholdEditor open={false} onClose={vi.fn()} thresholds={BASE_THRESHOLDS} onChange={vi.fn()} />);
        rerender(<ThresholdEditor open={true} onClose={vi.fn()} thresholds={BASE_THRESHOLDS} onChange={vi.fn()} />);

        expect(screen.getByDisplayValue("High")).toBeInTheDocument();
        expect(screen.queryByDisplayValue("Edited")).not.toBeInTheDocument();
    });
});
