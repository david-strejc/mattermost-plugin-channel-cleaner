import manifest from './manifest';

export default class Plugin {
    initialize(registry, store) {
        const {id} = manifest;
        
        // Create icon component
        const TrashIcon = () => (
            <i className="icon icon-trash-can-outline" style={{fontSize: 16}} />
        );
        
        // Register channel header button
        registry.registerChannelHeaderButtonAction(
            TrashIcon,
            (channel) => {
                // Show confirmation dialog
                const confirmed = window.confirm(
                    'WARNING: This will permanently delete all messages in this channel.\n\n' +
                    'Are you sure you want to clear all messages?'
                );
                
                if (confirmed) {
                    const textarea = document.getElementById('post_textbox');
                    const form = textarea ? textarea.closest('form') : null;
                    
                    if (textarea && form) {
                        // Save current value
                        const currentValue = textarea.value;
                        
                        // Set command text
                        textarea.value = '/clearchannel confirm';
                        
                        // Fire React's change event
                        const nativeInputValueSetter = Object.getOwnPropertyDescriptor(
                            window.HTMLTextAreaElement.prototype,
                            'value'
                        ).set;
                        nativeInputValueSetter.call(textarea, '/clearchannel confirm');
                        
                        const inputEvent = new Event('input', { bubbles: true });
                        textarea.dispatchEvent(inputEvent);
                        
                        // Submit form
                        setTimeout(() => {
                            // Find submit button or trigger form submit
                            const submitButton = form.querySelector('button[type="submit"]');
                            if (submitButton && !submitButton.disabled) {
                                submitButton.click();
                            } else {
                                // Try Enter key
                                const enterEvent = new KeyboardEvent('keypress', {
                                    key: 'Enter',
                                    code: 'Enter',
                                    which: 13,
                                    keyCode: 13,
                                    bubbles: true,
                                    cancelable: true,
                                });
                                textarea.dispatchEvent(enterEvent);
                            }
                            
                            // Reset value if command didn't submit
                            setTimeout(() => {
                                if (textarea.value === '/clearchannel confirm') {
                                    textarea.value = currentValue;
                                    nativeInputValueSetter.call(textarea, currentValue);
                                    textarea.dispatchEvent(new Event('input', { bubbles: true }));
                                }
                            }, 200);
                        }, 100);
                    } else {
                        alert('Could not find message input. Please type "/clearchannel confirm" manually.');
                    }
                }
            },
            'Clear all messages in this channel'
        );
    }
}

window.registerPlugin(manifest.id, new Plugin());