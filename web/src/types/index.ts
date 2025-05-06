export interface Module {
  name: string;
  description: string;
  versions: Version[];
}

export interface Version {
  moduleName: string;
  version: string;
  files: File[];
  dependencies: string[];
}

export interface File {
  path: string;
  content: string;
}

export interface Dependency {
  module: string;
  version: string;
} 