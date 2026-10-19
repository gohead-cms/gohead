/** A run of text that both sides share, or that only one side has. */
export interface DiffPart {
  kind: "same" | "added" | "removed";
  text: string;
}

/** Beyond this many token pairs the LCS table is too big: no inline diff. */
const MAX_CELLS = 400_000;

/** Splits text into words and the whitespace between them. */
function tokenize(text: string): string[] {
  return text.split(/(\s+)/).filter((t) => t !== "");
}

function push(parts: DiffPart[], kind: DiffPart["kind"], text: string) {
  const last = parts[parts.length - 1];
  if (last && last.kind === kind) last.text += text;
  else parts.push({ kind, text });
}

/**
 * A word-level diff of two strings (longest common subsequence), or null
 * when they are too long to compare quickly.
 */
export function wordDiff(before: string, after: string): DiffPart[] | null {
  const a = tokenize(before);
  const b = tokenize(after);

  // The shared start and end need no table.
  let start = 0;
  while (start < a.length && start < b.length && a[start] === b[start]) start++;
  let endA = a.length;
  let endB = b.length;
  while (endA > start && endB > start && a[endA - 1] === b[endB - 1]) {
    endA--;
    endB--;
  }
  const midA = a.slice(start, endA);
  const midB = b.slice(start, endB);
  if ((midA.length + 1) * (midB.length + 1) > MAX_CELLS) return null;

  // lcs[i][j]: length of the LCS of midA[i:] and midB[j:].
  const n = midA.length;
  const m = midB.length;
  const lcs: Uint32Array[] = Array.from({ length: n + 1 }, () => new Uint32Array(m + 1));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      lcs[i][j] = midA[i] === midB[j] ? lcs[i + 1][j + 1] + 1 : Math.max(lcs[i + 1][j], lcs[i][j + 1]);
    }
  }

  const parts: DiffPart[] = [];
  if (start > 0) push(parts, "same", a.slice(0, start).join(""));
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (midA[i] === midB[j]) {
      push(parts, "same", midA[i]);
      i++;
      j++;
    } else if (lcs[i + 1][j] >= lcs[i][j + 1]) {
      push(parts, "removed", midA[i++]);
    } else {
      push(parts, "added", midB[j++]);
    }
  }
  while (i < n) push(parts, "removed", midA[i++]);
  while (j < m) push(parts, "added", midB[j++]);
  if (endA < a.length) push(parts, "same", a.slice(endA).join(""));
  return parts;
}

/** Strings long enough to read better as an inline diff than side by side. */
export function isLongText(v: unknown): v is string {
  return typeof v === "string" && (v.length > 60 || v.includes("\n"));
}
