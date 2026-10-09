//go:build darwin && cgo
#import "darwin_desktop.h"

@implementation CPDesktop (Notifications)
-(void)prepareNotifications {
 if([NSBundle.mainBundle.bundleIdentifier isEqualToString:@"io.clipare.app"])[UNUserNotificationCenter currentNotificationCenter].delegate=self;
}
-(void)notifyPair:(NSString *)name {
 // CLI and synthetic UI tests never ask for notification authorization.
 if(![NSBundle.mainBundle.bundleIdentifier isEqualToString:@"io.clipare.app"])return;
 UNUserNotificationCenter *center=[UNUserNotificationCenter currentNotificationCenter];
 NSUInteger generation=++self.notificationGeneration;
 [center requestAuthorizationWithOptions:UNAuthorizationOptionAlert completionHandler:^(BOOL granted,NSError *error){
  dispatch_async(dispatch_get_main_queue(),^{
   if(!granted||self.closed||!self.pairIncoming||generation!=self.notificationGeneration)return;
   UNMutableNotificationContent *content=[[[UNMutableNotificationContent alloc]init]autorelease];content.title=@"Новое устройство хочет подключиться";content.body=name;content.categoryIdentifier=@"clipare.pairing";
   [center addNotificationRequest:[UNNotificationRequest requestWithIdentifier:@"clipare.pairing" content:content trigger:nil] withCompletionHandler:nil];
  });
 }];
}
-(void)clearPairNotification {
 self.pairIncoming=NO;
 self.notificationGeneration++;
 if(![NSBundle.mainBundle.bundleIdentifier isEqualToString:@"io.clipare.app"])return;
 UNUserNotificationCenter *center=[UNUserNotificationCenter currentNotificationCenter];[center removePendingNotificationRequestsWithIdentifiers:@[@"clipare.pairing"]];[center removeDeliveredNotificationsWithIdentifiers:@[@"clipare.pairing"]];
}
-(void)userNotificationCenter:(UNUserNotificationCenter *)center willPresentNotification:(UNNotification *)notification withCompletionHandler:(void (^)(UNNotificationPresentationOptions))completion {
 if(@available(macOS 11.0,*))completion(UNNotificationPresentationOptionBanner|UNNotificationPresentationOptionList);
 else {
  // Required on macOS 10.14–10.15; the replacement starts at macOS 11.
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
  completion(UNNotificationPresentationOptionAlert);
#pragma clang diagnostic pop
 }
}
-(void)userNotificationCenter:(UNUserNotificationCenter *)center didReceiveNotificationResponse:(UNNotificationResponse *)response withCompletionHandler:(void (^)(void))completion {
 dispatch_async(dispatch_get_main_queue(),^{if(!self.closed&&self.pairIncoming)[self show:CPPairing];completion();});
}
@end
