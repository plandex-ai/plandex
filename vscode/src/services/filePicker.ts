import * as vscode from 'vscode';
import * as path from 'path';
import { FilePickerItem } from '../types';
import { executeCommand } from '../extension';

export class FilePickerService {
  private quickPick: vscode.QuickPick<FilePickerItem>;

  constructor() {
    this.quickPick = vscode.window.createQuickPick<FilePickerItem>();
    this.quickPick.onDidAccept(() => this.handleQuickPickAccept());
    this.quickPick.onDidHide(() => this.quickPick.dispose());
  }


  private async handleQuickPickAccept(): Promise<void> {
    const selectedItem = this.quickPick.selectedItems[0];
    if (!selectedItem) return;

    const activeEditor = vscode.window.activeTextEditor;
    if (!activeEditor) return;

    const pdxFilePath = activeEditor.document.uri.fsPath;
    const cwd = path.dirname(pdxFilePath);

    executeCommand({
      command: 'plandex',
      args: ['load', selectedItem.path],
      cwd
    });

    this.quickPick.hide();
  }



  public async showFilePicker(): Promise<void> {
    const activeEditor = vscode.window.activeTextEditor;
    if (!activeEditor) return;

    const workspaceFolder = vscode.workspace.getWorkspaceFolder(activeEditor.document.uri);
    if (!workspaceFolder) return;

    const pdxFilePath = activeEditor.document.uri.fsPath;
    const pdxFileDir = path.dirname(pdxFilePath);

    const files = await vscode.workspace.findFiles('**/*', '**/node_modules/**');
    
    const items: FilePickerItem[] = files
      .filter(file => file.fsPath !== pdxFilePath) // Don't include current file
      .map(file => {
        const filePath = file.fsPath;
        const relativePath = path.relative(pdxFileDir, filePath);
        
        return {
          label: relativePath,
          description: path.basename(filePath),
          path: filePath,
          relativePath
        };
      });

    this.quickPick.items = items;
    this.quickPick.placeholder = 'Select a file to add to context';
    this.quickPick.show();
  }
}
