import { createContext, useContext } from 'react';

interface SidebarContextType {
  isCollapsed: boolean;
  toggleSidebar: () => void;
}

// Create a context to hold the sidebar's state
export const SidebarContext = createContext<SidebarContextType | undefined>(undefined);

// Create a custom hook for easy access to the context
export const useSidebar = () => {
  const context = useContext(SidebarContext);
  if (!context) {
    throw new Error('useSidebar must be used within a SidebarProvider');
  }
  return context;
};
