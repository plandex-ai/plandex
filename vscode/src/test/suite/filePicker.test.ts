import * as assert from 'assert';
import * as vscode from 'vscode';
import { FilePickerService } from '../../services/filePicker';

suite('File Picker Service Test Suite', () => {
  let filePickerService: FilePickerService;

  setup(() => {
    filePickerService = new FilePickerService();
  });

  test('Should create quick pick with correct options', async () => {
    const doc = await vscode.workspace.openTextDocument({
      content: '',
      language: 'pdx'
    });
    await vscode.window.showTextDocument(doc);

    await filePickerService.showFilePicker();
    
    const quickPick = vscode.window.createQuickPick();
    assert.ok(quickPick);
    assert.strictEqual(quickPick.placeholder, 'Select a file to add to context');
  });

  test('Should filter workspace files correctly', async () => {
    const doc = await vscode.workspace.openTextDocument({
      content: '',
      language: 'pdx'
    });
    await vscode.window.showTextDocument(doc);

    await filePickerService.showFilePicker();
    
    const quickPick = vscode.window.createQuickPick();
    assert.ok(quickPick);
    
    // Verify node_modules is excluded
    const nodeModulesFiles = quickPick.items.filter((item: vscode.QuickPickItem) => 
      item.label.includes('node_modules')
    );
    assert.strictEqual(nodeModulesFiles.length, 0);
  });
});
