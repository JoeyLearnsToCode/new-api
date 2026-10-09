export interface Project {
  id: string;
  folderId: string | null;
  name: string;
  createdAt: number;
  updatedAt: number;
  tags?: {
    model?: string;
    platform?: string;
    type?: string;
  };
}
