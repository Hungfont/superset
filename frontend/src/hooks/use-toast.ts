import { toast as sonnerToast } from "sonner";

type ToastOptions = {
  description?: string;
};

export function useToast() {
  return {
    toast: sonnerToast,
    success: (title: string, options?: ToastOptions) => sonnerToast.success(title, options),
    error: (title: string, options?: ToastOptions) => sonnerToast.error(title, options),
    info: (title: string, options?: ToastOptions) => sonnerToast.info(title, options),
    warning: (title: string, options?: ToastOptions) => sonnerToast.warning(title, options),
    dismiss: sonnerToast.dismiss,
  };
}
