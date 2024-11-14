import * as vscode from 'vscode';
import * as path from 'path';
import { executeCommand } from '../extension';

export class TellPlandexService {
  public async executeTellPlandex(): Promise<void> {
    const activeEditor = vscode.window.activeTextEditor;
    if (!activeEditor) return;

    const pdxFilePath = activeEditor.document.uri.fsPath;
    const cwd = path.dirname(pdxFilePath);

    // Initialize new plan if needed
    executeCommand({
      command: 'plandex',
      args: ['new'],
      cwd
    });
    
    // Tell Plandex about the current file
    executeCommand({
      command: 'plandex',
      args: ['tell', '-f', pdxFilePath],
      cwd
    });
  }
}
