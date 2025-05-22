/* global document$:readonly */
document$.subscribe(() => {
  document.querySelectorAll("pre > code").forEach((code) => {
    const wrapper = code.closest("div.language-console");

    if (!wrapper) return;

    const button = wrapper.querySelector("button.md-clipboard");
    if (!button) return;

    button.addEventListener(
      "mouseenter",
      () => {
        // Only set data-copy once
        if (code.hasAttribute("data-copy")) return;

        const text = code.textContent.trimEnd();

        // Merge continuation lines that end with '\', and remove '$' from prompts
        let lines = [];
        let mergeNext = false;

        text.split("\n").forEach((line) => {
          // If we need to merge the current line with the next one
          if (mergeNext) {
            // Merge the line with the previous one and reset the flag
            lines[lines.length - 1] += " " + line.trimStart();
            if (lines[lines.length - 1].endsWith(" \\")) {
              lines[lines.length - 1] = lines[lines.length - 1].slice(0, -2);
            } else if (lines[lines.length - 1].endsWith("\\")) {
              lines[lines.length - 1] = lines[lines.length - 1].slice(0, -1);
            } else {
              mergeNext = false;
            }
          }

          // If the line starts with '$' and ends with '\' (continuation line)
          if (line.startsWith("$ ")) {
            if (line.endsWith(" \\")) {
              // Remove the '$' and backslash, and mark that we need to merge with the next line
              lines.push(line.slice(2, -2)); // Remove "$ " and the trailing "\"
              mergeNext = true;
            } else if (line.endsWith("\\")) {
              lines.push(line.slice(2, -1)); // Remove "$ " and the trailing " \"
              mergeNext = true;
            } else {
              // If it's a prompt line without continuation, just remove the '$'
              lines.push(line.slice(2));
            }
          }
        });

        const cleaned = lines.join("\n");

        code.setAttribute("data-copy", cleaned);
      },
      { once: true },
    );
  });
});
