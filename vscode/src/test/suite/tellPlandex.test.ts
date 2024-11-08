import * as assert from 'assert';
import * as vscode from 'vscode';
import { TellPlandexService } from '../../services/tellPlandex';

suite('Tell Plandex Service Test Suite', () => {
  let tellPlandexService: TellPlandexService;

  setup(() => {
    tellPlandexService = new TellPlandexService();
  });

  test('Should execute tell plandex commands', async () => {
    const doc = await vscode.workspace.openTextDocument({
      content: '',
      language: 'pdx'
    });
    await vscode.window.showTextDocument(doc);

    let commandsExecuted = 0;
    const disposable = vscode.commands.registerCommand('workbench.action.terminal.sendSequence', () => {
      commandsExecuted++;
      return Promise.resolve();
    });

    try {
      await tellPlandexService.executeTellPlandex();
      assert.strictEqual(commandsExecuted, 2); // new and tell commands
    } finally {
      disposable.dispose();
    }
  });
});
