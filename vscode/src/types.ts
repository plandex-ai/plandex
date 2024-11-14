import * as vscode from 'vscode';

export interface PlanCreator {
  name: string;
  email: string;
  timestamp: string;
}

export type PlanStatus = 'draft' | 'in-progress' | 'completed' | 'archived';

export interface PlanFrontmatter {
  name: string;
  id: string;
  creator: PlanCreator;
  lastUpdatedBy: PlanCreator;
  collaborators: PlanCreator[];
  status: PlanStatus;
}

export interface PlanDocument {
  frontmatter: PlanFrontmatter;
  contexts: string[];
  prompts: string[];
}

export interface TerminalCommand {
  command: string;
  args: string[];
  cwd: string;
}

export interface FilePickerItem extends vscode.QuickPickItem {
  path: string;
  relativePath: string;
}
