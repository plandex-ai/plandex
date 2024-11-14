import * as vscode from "vscode";
import { parse } from "yaml";
import { FilePickerService } from "./services/filePicker";
import { TellPlandexService } from "./services/tellPlandex";
import type { PlanDocument, PlanFrontmatter, TerminalCommand } from "./types";

let terminal: vscode.Terminal | undefined;

export function getOrCreateTerminal(cwd: string): vscode.Terminal {
  if (!terminal || terminal.exitStatus !== undefined) {
    terminal = vscode.window.createTerminal({
      name: "Plandex",
      cwd,
    });
  }
  return terminal;
}

export function executeCommand(command: TerminalCommand): void {
  const terminal = getOrCreateTerminal(command.cwd);
  terminal.show();
  terminal.sendText(`${command.command} ${command.args.join(" ")}`);
}

export function getPlanDocument(
  document: vscode.TextDocument
): PlanDocument | undefined {
  try {
    const text = document.getText();
    const frontmatterMatch = text.match(/^---\n([\s\S]*?)\n---/);

    if (!frontmatterMatch) {
      return undefined;
    }

    const frontmatter = parse(frontmatterMatch[1]) as PlanFrontmatter;

    const contextsMatch = text.match(/<Contexts>([\s\S]*?)<\/Contexts>/);
    const promptsMatch = text.match(/<Prompts>([\s\S]*?)<\/Prompts>/);

    return {
      frontmatter,
      contexts: contextsMatch ? contextsMatch[1].trim().split("\n") : [],
      prompts: promptsMatch ? promptsMatch[1].trim().split("\n") : [],
    };
  } catch (error) {
    console.error("Error parsing plan document:", error);
    return undefined;
  }
}

export function activate(context: vscode.ExtensionContext): void {
  console.log("Plandex extension is now active");

  const filePickerService = new FilePickerService();
  const tellPlandexService = new TellPlandexService();

  // Register the file picker command
  context.subscriptions.push(
    vscode.commands.registerTextEditorCommand("plandex.showFilePicker", () => {
      filePickerService.showFilePicker();
    })
  );


  console.log("registered file picker command");

  // Register the Tell Plandex command
  context.subscriptions.push(
    vscode.commands.registerCommand("plandex.tellPlandex", () => {
      tellPlandexService.executeTellPlandex();
    })
  );

  console.log("registered tell plandex command");

  // Create status bar item for Tell Plandex
  const tellPlandexButton = vscode.window.createStatusBarItem(
    vscode.StatusBarAlignment.Right,
    100
  );

  tellPlandexButton.command = "plandex.tellPlandex";
  tellPlandexButton.text = "$(play) Tell Plandex";
  tellPlandexButton.tooltip = "Tell Plandex about the current file";
  tellPlandexButton.color = "#89D185";

  context.subscriptions.push(tellPlandexButton);

  // Show/hide the button based on active editor
  context.subscriptions.push(
    vscode.window.onDidChangeActiveTextEditor((editor) => {
      if (editor?.document.languageId === "pdx") {
        tellPlandexButton.show();
      } else {
        tellPlandexButton.hide();
      }
    })
  );

  // Initial visibility
  const activeEditor = vscode.window.activeTextEditor;
  const languageId = activeEditor?.document.languageId;
  console.log("active editor language id", languageId);
  if (languageId === "pdx") {
    tellPlandexButton.show();
  }
}

export function deactivate(): void {
  if (terminal) {
    terminal.dispose();
  }
}
