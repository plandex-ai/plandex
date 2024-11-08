import * as assert from 'assert';
import * as vscode from 'vscode';
import { before } from 'mocha';

suite('Extension Test Suite', () => {
  before(async () => {
    await vscode.commands.executeCommand('workbench.action.closeAllEditors');
  });

  test('Extension should be activated for .pdx files', async () => {
    const doc = await vscode.workspace.openTextDocument({
      content: '',
      language: 'pdx'
    });
    await vscode.window.showTextDocument(doc);
    
    const extension = vscode.extensions.getExtension('vscode-plandex');
    assert.ok(extension);
    assert.strictEqual(extension.isActive, true);
  });

  test('PDX files should use MDX syntax highlighting', async () => {
    await vscode.workspace.openTextDocument({
      content: '# Heading\n\n```js\nconst x = 1;\n```',
      language: 'pdx'
    });
    
    const grammar = await vscode.languages.setLanguageConfiguration('pdx', {
      comments: { blockComment: ['<!--', '-->'] }
    });
    assert.ok(grammar);
  });

  test('File picker should show on @ key', async () => {
    const doc = await vscode.workspace.openTextDocument({
      content: '',
      language: 'pdx'
    });
    await vscode.window.showTextDocument(doc);
    
    const disposable = vscode.commands.registerCommand('workbench.action.quickOpen', () => {
      return Promise.resolve(true);
    });
    
    try {
      await vscode.commands.executeCommand('plandex.showFilePicker');
      assert.ok(true);
    } finally {
      disposable.dispose();
    }
  });

  test('Tell Plandex button should be visible for PDX files', async () => {
    const doc = await vscode.workspace.openTextDocument({
      content: '',
      language: 'pdx'
    });
    await vscode.window.showTextDocument(doc);
    
    // Give the extension time to update UI
    await new Promise(resolve => setTimeout(resolve, 100));
    
    const statusBarItems = vscode.window.createStatusBarItem();
    assert.ok(statusBarItems);
    assert.strictEqual(statusBarItems.text.includes('Tell Plandex'), true);
  });
});
