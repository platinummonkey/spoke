export interface SourceInfo {
  repository: string;
  commit_sha: string;
  branch: string;
}

export interface Module {
  name: string;
  description: string;
  versions: Version[];
}

export interface Version {
  module_name: string;
  version: string;
  files: File[];
  created_at: string;
  dependencies?: string[];
  source_info: SourceInfo;
}

export interface File {
  path: string;
  content: string;
}

export interface Dependency {
  module: string;
  version: string;
} 