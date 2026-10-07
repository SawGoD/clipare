#import <AppKit/AppKit.h>
#include "../../internal/ui/native_darwin.m"

// Capture paste without reading or changing the user's clipboard.
@interface PasteRecorder : NSTextView
@property BOOL pasted;
@end
@implementation PasteRecorder
- (void)paste:(id)sender {self.pasted=YES;}
- (BOOL)validateMenuItem:(NSMenuItem *)item {return YES;}
@end

int main(void) { @autoreleasepool {
 clipare_init();
 PasteRecorder *recorder=[[PasteRecorder alloc] initWithFrame:NSMakeRect(0,0,50,30)];
 [window.contentView addSubview:recorder];
 [window makeKeyAndOrderFront:nil];[NSApp activateIgnoringOtherApps:YES];
 [window makeFirstResponder:recorder];
 for(int i=0;i<20;i++){clipare_poll();}
 NSMenu *edit=[[NSApp.mainMenu itemAtIndex:1] submenu];
 NSMenuItem *paste=[edit itemAtIndex:3];
 if(paste.action!=@selector(paste:)||![paste.keyEquivalent isEqualToString:@"v"]||paste.target!=nil){clipare_close();fprintf(stderr,"invalid paste menu\n");return 1;}
 NSEvent *event=[NSEvent keyEventWithType:NSEventTypeKeyDown location:NSZeroPoint modifierFlags:NSEventModifierFlagCommand timestamp:0 windowNumber:window.windowNumber context:nil characters:@"v" charactersIgnoringModifiers:@"v" isARepeat:NO keyCode:9];
 BOOL handled=[NSApp.mainMenu performKeyEquivalent:event];
 clipare_close();
 if(!handled||!recorder.pasted){fprintf(stderr,"Command+V did not reach the field editor\n");return 2;}
 puts("PASS: Command+V reaches paste:; clipboard untouched");return 0;
} }
