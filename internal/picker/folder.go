package picker

// PickFolder asks for a folder and returns its path, or "" when the dialog is cancelled.
func (s *Service) PickFolder(title string) (string, error) {
	d := s.App.Dialog.OpenFile().
		SetTitle(title).
		CanChooseDirectories(true).
		CanChooseFiles(false)
	if w := s.App.Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
	return d.PromptForSingleSelection()
}
